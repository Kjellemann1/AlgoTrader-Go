package src

import (
  "fmt"
  "log"
  "regexp"
  "errors"
  "strconv"
  "sync"
  "time"
  "github.com/gorilla/websocket"
  "github.com/valyala/fastjson"
  "github.com/shopspring/decimal"
  "github.com/Kjellemann1/AlgoTrader-Go/src/constant"
  "github.com/Kjellemann1/AlgoTrader-Go/src/order"
  "github.com/Kjellemann1/AlgoTrader-Go/src/util/handlelog"
  "github.com/Kjellemann1/AlgoTrader-Go/src/util/backoff"
)

type OrderUpdate struct {
  Event          string
  AssetClass     string
  StratName      string
  Side           *string
  Symbol         *string
  AssetQty       *decimal.Decimal
  FillTime       *time.Time
  FilledAvgPrice *float64
}

type Account struct {
  conn *websocket.Conn
  parser fastjson.Parser
  db_chan chan *Query
  assets map[string]map[string]*Asset
  // TODO stock and crypto cleared for shutdown
}

func NewAccount(assets map[string]map[string]*Asset, db_chan chan *Query) *Account {
  a := &Account{
    parser: fastjson.Parser{},
    assets: assets,
    db_chan: db_chan,
  }
  return a
}

func (a *Account) onAuth(element *fastjson.Value) {
  msg := string(element.Get("data").GetStringBytes("status"))
  if msg == "authorized" {
    log.Println("[ OK ]\tAuthenticated with account websocket")
  } else {
    log.Panicln("[ ERROR ]\tAuthorization with account websocket failed")
  }
}

func (a *Account) messageHandler(message []byte) {
  parsed_msg, err := a.parser.ParseBytes(message)
  if err != nil {
    log.Println("Error parsing json: ", err)
    // TODO: Implement error handling
  }
  // Handle each message based on the "T" field
  message_type := string(parsed_msg.GetStringBytes("stream"))
  switch message_type {
    case "authorization":
      a.onAuth(parsed_msg)
    case "trade_updates":
      update := a.updateParser(&ParsedMessage{parsed_msg})
      if update == nil {
        return
      }
      a.orderUpdateHandler(update)

    case "listening":
      log.Println("[ OK ]\tListening to order updates")
  }
}

func (a *Account) connect(initial *bool) (err error) {
  conn, response, err := websocket.DefaultDialer.Dial("wss://paper-api.alpaca.markets/stream", nil)
  if err != nil && *initial {
    log.Panicf("[ ERROR ]\tCould not connect to account websocket: %s\n  -> %+v", err, response)
  } else if err != nil {
    log.Println("[ ERROR ]\tCould not connect to account websocket: ", err)
    return
  }
  err = conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf(`{"action":"auth","key":"%s","secret":"%s"}`, constant.KEY, constant.SECRET)))
  if err != nil && *initial {
    log.Panicf("[ ERROR ]\tSending connection message to account websocket failed: %s", err)
  } else if err != nil {
    log.Println("[ ERROR ]\tSending connection message to account websocket failed: ", err)
    return
  }
  _, message, err := conn.ReadMessage()
  if err != nil && *initial {
    log.Panicf(
      "[ ERROR ]\tReading connection message from account websocket failed\n" +
      "  -> Error: %s\n" +
      "  -> Message: %s",
      err, message,
    )
  } else if err != nil {
    log.Println("[ ERROR ]\tReading connection message from account websocket failed: ", err)
    return
  }
  a.messageHandler(message)
  err = conn.WriteMessage(websocket.TextMessage, []byte(`{"action":"listen","data":{"streams":["trade_updates"]}}`))
  if err != nil && *initial {
    log.Panicf(
      "[ ERROR ]\tSending listen message to account websocket failed\n" +
      "  -> Error: %s",
      err,
    )
  } else if err != nil {
    log.Println("[ ERROR ]\tSending listen message to account websocket failed: ", err)
    return
  }
  a.conn = conn
  *initial = false
  return
}

func (a *Account) PingPong() {
  if err := a.conn.SetReadDeadline(time.Now().Add(60 * time.Second)); err != nil {
    handlelog.Warning(err)
  }
  a.conn.SetPongHandler(func(string) error {
    a.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
    log.Println("[ Account ]\t<< PONG")
    return nil
  })

  go func() {
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()
    for {
      select {
      case <-ticker.C:
        if err := a.conn.WriteControl(websocket.PingMessage, []byte("ping"), time.Now().Add(5 * time.Second)); err != nil { handlelog.Warning(err)
          a.conn.Close()
          return
        } else {
          log.Println("[ Account ]\tPING >>")
        }
      }
    }
  }()
}

func (a *Account) listen() {
  defer a.conn.Close()
  for {
    _, message, err := a.conn.ReadMessage()
    if err != nil {
      if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway) {
        handlelog.Warning(err)
        return
      } else if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
        handlelog.Warning(err)
        return
      } else {
        NNP.NoNewPositionsTrue("Account.listen")
        handlelog.Error(err, "Message", string(message), "NoNewPositions", NNP.Flag)
        continue
      }
    }
    a.messageHandler(message)
  }
}

func (a *Account) Start(wg *sync.WaitGroup) {
  defer wg.Done()
  backoff_sec := 5
  retries := 0
  initial := true
  for {
    err := a.connect(&initial)
    if err != nil {
      if retries < 5 {
        handlelog.Error(err, "Retries", retries)
      } else {
        handlelog.Error(err, "MAXIMUM NUMBER OF RETRIES REACHED", retries, "CLOSING ALL POSITIONS AND SHUTTING DOWN", "...")
        order.CloseAllPositions(2, 0)
        log.Panic("SHUTTING DOWN")
      }
      backoff.Backoff(&backoff_sec)
      retries++
      continue
    }
    backoff_sec = 5
    retries = 0
    a.PingPong()
    a.listen()
    // TODO: On reconnect:
    //   -> Check that positions on the server are the same as locally
    //     -> Make sure we didn't miss any updates
    //   -> Make sure there are no panics on reconnect
  }
}

func calculatePositionQty(p *Position, a *Asset, u *OrderUpdate) {
  a.Rwm.Lock()
  defer a.Rwm.Unlock()
  var position_change decimal.Decimal = (*u.AssetQty).Sub(a.AssetQty)
  p.Qty = p.Qty.Add(position_change)
  a.AssetQty = *u.AssetQty
  if !a.SumPosQtyEqAssetQty() {
    handlelog.Error(
      errors.New("Sum of position qty not equal to asset qty"),
      "Asset", a.AssetQty, "Position", p.Qty, "OrderUpdate", u,
      "CLOSING ALL POSITIONS AND SHUTTING DOWN", "...",
    )
    order.CloseAllPositions(2, 0)
    log.Fatal("SHUTTING DOWN")
  }
}

type ParsedMessage struct {
  *fastjson.Value
}

func (p *ParsedMessage) handleNil(id string, shutdown bool) {
    if shutdown {
      handlelog.Error(
        errors.New(fmt.Sprintf("%s not in trade update", id)),
        "ID", id, "Parsed message", p.Value,
        "CLOSING ALL POSITIONS AND SHUTTING DOWN", "...",
      )
      order.CloseAllPositions(2, 0)
      log.Panicln("SHUTTING DOWN")
    }
    handlelog.Warning(
      errors.New(fmt.Sprintf("%s not in trade update", id)),
      "ID", id, "Parsed message", p.Value,
    )
}

func (p *ParsedMessage) getData(id string, shutdown bool) string {
  bytes := p.Get("data").GetStringBytes(id)
  if bytes == nil {
    p.handleNil(id, shutdown)
    return ""
  }
  return string(bytes)
}

func (p *ParsedMessage) getOrder(id string, shutdown bool) string {
  bytes := p.Get("data").Get("order").GetStringBytes(id)
  if bytes == nil {
    p.handleNil(id, shutdown)
    return ""
  }
  return string(bytes)
}

func (p *ParsedMessage) grepStratName(orderID string) (string, error) {
  pattern := `strat\[(.*?)\]`
  re := regexp.MustCompile(pattern)
  match := re.FindStringSubmatch(orderID)
  if match != nil && len(match) > 1 {
    return match[1], nil
  }
  return "", errors.New("")
}

func (p *ParsedMessage) getEvent() (event string, err error) {
  // Only handle fill, partial_fill and canceled events.
  // Other events are likely not relevant. https://alpaca.markets/docs/api-documentation/api-v2/streaming/
  event = p.getData("event", true)
  if event != "fill" && event != "partial_fill" && event != "canceled" {
    return "", errors.New("Event not fill, partial_fill or canceled")
  }
  return
}

func (p *ParsedMessage) getAssetClass() (asset_class string) {
  asset_class = p.getData("asset_class", true)
  if asset_class == "us_equity" {
    asset_class = "stock"
  }
  return
}

func (p *ParsedMessage) getSymbol() *string {
  symbol := p.getData("symbol", true)
  if symbol != "" {
    return &symbol
  }
  return nil
}

func (p *ParsedMessage) getStratName() (string, error) {
  id := p.getOrder("client_order_id", true)
  strat_name, err := p.grepStratName(id)
  return strat_name, err
}

func (p *ParsedMessage) getSide() *string {
  side := p.getOrder("side", false)
  if side != "" {
    return &side
  }
  return nil
}

func (p *ParsedMessage) getAssetQty() *decimal.Decimal {
  asset_qty := p.getOrder("qty", false)
  if asset_qty != "" {
    asset_qty_dec, err := decimal.NewFromString(string(asset_qty))
    if err != nil {
      handlelog.Error(err, "Asset qty", asset_qty, "CLOSING ALL POSITIONS AND SHUTTING DOWN", "...")
      order.CloseAllPositions(2, 0)
      log.Panicln("SHUTTING DOWN")
    }
    return &asset_qty_dec
  }
  return nil
}

func (p *ParsedMessage) getFillTime() *time.Time {
  fill_time_str := p.getOrder("filled_at", false)
  if fill_time_str != "" {
    fill_time, err := time.Parse(time.RFC3339, fill_time_str)
    if err != nil {
      handlelog.Warning(err, "Fill time", fill_time_str)
      return nil
    }
    return &fill_time
  }
  return nil
}

func (p *ParsedMessage) getFilledAvgPrice() *float64 {
  filled_avg_price_str := p.getOrder("filled_avg_price", false)
  if filled_avg_price_str != "" {
    filled_avg_price, err := strconv.ParseFloat(filled_avg_price_str, 8)
    if err != nil {
      handlelog.Warning(err, "Filled avg price", filled_avg_price_str)
      return nil
    }
    return &filled_avg_price
  }
  return nil
}

func (a *Account) updateParser(p *ParsedMessage) *OrderUpdate {
  event, err := p.getEvent()
  if err != nil {
    return nil
  }
  strat_name, err := p.getStratName()
  if err != nil {
    return nil
  }
  return &OrderUpdate {
    Event:            event,
    StratName:        strat_name,
    AssetClass:       p.getAssetClass(),
    Side:             p.getSide(),
    Symbol:           p.getSymbol(),
    AssetQty:         p.getAssetQty(),
    FillTime:         p.getFillTime(),
    FilledAvgPrice:   p.getFilledAvgPrice(),
  }
}

func (a *Account) orderUpdateHandler(u *OrderUpdate) {
  var asset = a.assets[u.AssetClass][*u.Symbol]
  var pos *Position = asset.Positions[u.StratName]
  if pos == nil {
    log.Panicf("Position nil: %s", *u.Symbol)
  }
  pos.Rwm.Lock()
  defer pos.Rwm.Unlock()
  // Update AssetQty
  if u.AssetQty != nil {
    calculatePositionQty(pos, asset, u)
  }
  // Open order logic
  if pos.OpenOrderPending {
    if u.FilledAvgPrice != nil {
      pos.OpenFilledAvgPrice = *u.FilledAvgPrice
    }
    if u.FillTime != nil {
      pos.OpenFillTime = *u.FillTime
    }
    if u.Event == "fill" || u.Event == "canceled" {
      if pos.Qty.IsZero() {
        asset.RemovePosition(u.StratName)
      } else {
        a.db_chan <-pos.LogOpen(u.StratName)
      }
      pos.OpenOrderPending = false
    }
  // Close order logic
  } else if pos.CloseOrderPending {
    if u.FilledAvgPrice != nil {
      pos.CloseFilledAvgPrice = *u.FilledAvgPrice
    }
    if u.FillTime != nil {
      pos.CloseFillTime = *u.FillTime
    }
    if u.Event == "fill" || u.Event == "canceled" {
      a.db_chan <-pos.LogClose(u.StratName)
      if pos.Qty.IsZero() {
        if pos.CloseFilledAvgPrice == 0 {  // Remove
          log.Printf(
            "[ ERROR ]\tLogClose called with zero filled avg price"+
            "  -> OrderUpdate: %+v\n"+
            "  -> Position before removal: {Symbol: %s, CloseFillTime: %v, Qty: %s, CloseFilledAvgPrice: %f, PositionID: %s}\n",
            *u, pos.Symbol, pos.CloseFillTime, pos.Qty.String(), pos.CloseFilledAvgPrice, pos.PositionID,
          )
        }
        asset.RemovePosition(u.StratName)
      } else {
        pos.CloseOrderPending = false
        asset.Close("IOC", u.StratName)
      }
    }
  }
}
