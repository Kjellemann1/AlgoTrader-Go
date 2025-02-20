package src

import (
  "fmt"
  "log"
  "regexp"
  "errors"
  "strconv"
  "sync"
  "time"
  "context"
  "net/http"

  "github.com/gorilla/websocket"
  "github.com/valyala/fastjson"
  "github.com/shopspring/decimal"

  "github.com/Kjellemann1/AlgoTrader-Go/src/constant"
  "github.com/Kjellemann1/AlgoTrader-Go/src/order"
  "github.com/Kjellemann1/AlgoTrader-Go/src/util/handlelog"
  "github.com/Kjellemann1/AlgoTrader-Go/src/util/backoff"
)

type ParsedMessage struct {
  *fastjson.Value
}

type OrderUpdate struct {
  Event          *string
  AssetClass     *string
  StratName      *string
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
    handlelog.Error(err)
    return
  }
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

func (a *Account) connect() (err error) {
  var response *http.Response
  a.conn, response, err = websocket.DefaultDialer.Dial(
    "wss://paper-api.alpaca.markets/stream", nil,
  )
  if err != nil {
    log.Println("Response", response.Body)
    return err
  }
  err = a.conn.WriteMessage(websocket.TextMessage, []byte(
    fmt.Sprintf(`{"action":"auth","key":"%s","secret":"%s"}`, constant.KEY, constant.SECRET),
  ))
  if err != nil {
    return
  }
  _, message, err := a.conn.ReadMessage()
  if err != nil {
    log.Println("Message", string(message))
    return
  }
  a.messageHandler(message)
  err = a.conn.WriteMessage(websocket.TextMessage,
    []byte(`{"action":"listen","data":{"streams":["trade_updates"]}}`),
  )
  if err != nil {
    return
  }
  return nil
}

func (a *Account) ordersPending() bool {
  for _, class := range a.assets {
    for _, asset := range class {
      for _, position := range (*asset).Positions {
        if position.OpenOrderPending || position.CloseOrderPending {
          return true
        }
      }
    }
  }
  return false
}

func (a *Account) checkOrdersPending(ctx context.Context) {
  ticker := time.NewTicker(5 * time.Second)
  defer ticker.Stop()
  <-ctx.Done()
  for range ticker.C {
    if a.ordersPending() {
      continue
    } else {
      a.db_chan <- nil
      a.conn.Close()
      return
    }
  }
}

func (a *Account) listen(ctx context.Context) {
  for {
    _, message, err := a.conn.ReadMessage()
    if err != nil {
      select {
      case <-ctx.Done():
        return
      default:
        handlelog.Error(err)
        a.conn.Close()
        return
      }
    }
    a.messageHandler(message)
  }
}

func (a *Account) PingPong(ctx context.Context) {
  if err := a.conn.SetReadDeadline(time.Now().Add(60 * time.Second)); err != nil {
    handlelog.Warning(err)
  }

  a.conn.SetPongHandler(func(string) error {
    a.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
    return nil
  })

  ticker := time.NewTicker(30 * time.Second)
  defer ticker.Stop()
  for {
    select {
    case <-ctx.Done():
      return
    case <-ticker.C:
      if err := a.conn.WriteControl(
        websocket.PingMessage, []byte("ping"), 
        time.Now().Add(5 * time.Second)); err != nil {
        handlelog.Warning(err)
        return
      }
    }
  }
}

func (a *Account) Start(wg *sync.WaitGroup, ctx context.Context) {
  defer wg.Done()
  go a.checkOrdersPending(ctx)
  backoff_sec := 5
  retries := 0
  initial := true

  for {
    select {
    case <-ctx.Done():
      return
    default:
      if err := a.connect() ; err != nil {
        if initial {
          panic(err.Error())
        }
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

      initial = false
      backoff_sec = 5
      retries = 0

      go a.listen(ctx)
      a.PingPong(ctx)
      a.conn.Close()
      // TODO: On reconnect, check that positions on the server are the same as locally
      //   -> Make sure we didn't miss any updates
    }
  }
}

func (p *ParsedMessage) getEvent() *string {
  // Shutdown if nil
  // Only handle fill, partial_fill and canceled events.
  // Other events are likely not relevant. https://alpaca.markets/docs/api-documentation/api-v2/streaming/
  event := p.Get("data").GetStringBytes("event")
  if event == nil {
    handlelog.Error(
      errors.New("EVENT NOT IN TRADE UPDATE"), "Parsed message", p.String(),
      "CLOSING ALL POSITIONS AND SHUTTING DOWN", "...",
    )
    order.CloseAllPositions(2, 0)
    log.Panicln("SHUTTING DOWN")
  }
  event_str := string(event)
  if event_str != "fill" && event_str != "partial_fill" && event_str != "canceled" {
    return nil
  }
  return &event_str
}

func (p *ParsedMessage) getStratName() *string {
  // Return if nil 
  position_id := p.Get("data").Get("order").GetStringBytes("client_order_id")  // client_order_id == PositionID
  if position_id == nil {
    handlelog.Warning(errors.New("PositionID not found in order update"), nil)
    return nil
  }
  position_id_str := string(position_id)
  pattern := `strat\[(.*?)\]`
  re := regexp.MustCompile(pattern)
  match := re.FindStringSubmatch(position_id_str)
  if match != nil && len(match) > 1 {
    return &match[1]
  }
  return nil
}

func (p *ParsedMessage) getAssetClass() *string {
  // Shutdown if nil
  asset_class := p.Get("data").Get("order").GetStringBytes("asset_class")
  if asset_class == nil {
    handlelog.Error(
      errors.New("ASSET CLASS NOT IN TRADE UPDATE"), "Parsed message", p.String(),
      "CLOSING ALL POSITIONS AND SHUTTING DOWN", "...",
    )
    order.CloseAllPositions(2, 0)
    log.Panicln("SHUTTING DOWN")
  }
  asset_class_str := string(asset_class)
  if asset_class_str == "us_equity" {
    asset_class_str = "stock"
  }
  return &asset_class_str
}

func (p *ParsedMessage) getSymbol() *string {
  // Return if nil
  symbol := p.Get("data").Get("order").GetStringBytes("symbol")
  if symbol == nil {
    handlelog.Error(
      errors.New("SYMBOL NOT IN TRADE UPDATE"), "Parsed message", p.String(),
      "CLOSING ALL POSITIONS AND SHUTTING DOWN", "...",
    )
    order.CloseAllPositions(2, 0)
    log.Panicln("SHUTTING DOWN")
  }
  symbol_str := string(symbol)
  return &symbol_str
}

func (p *ParsedMessage) getSide() *string {
  side := p.Get("data").Get("order").GetStringBytes("side")
  if side == nil {
    return nil
  }
  side_str := string(side)
  return &side_str
}

func (p *ParsedMessage) getAssetQty() *decimal.Decimal {
  // Return if nil, shutdown if fails
  asset_qty := p.Get("data").GetStringBytes("position_qty")
  if asset_qty == nil {
    return nil
  }
  asset_qty_dec, err := decimal.NewFromString(string(asset_qty))
  if err != nil {
    handlelog.Error(err, "Asset qty", asset_qty, "CLOSING ALL POSITIONS AND SHUTTING DOWN", "...")
    order.CloseAllPositions(2, 0)
    log.Panicln("SHUTTING DOWN")
  }
  return &asset_qty_dec
}

func (p *ParsedMessage) getFillTime() *time.Time {
  fill_time := p.Get("data").Get("order").GetStringBytes("filled_at")
  if fill_time == nil {
    return nil
  }
  fill_time_t, err := time.Parse(time.RFC3339, string(fill_time))
  if err != nil {
    handlelog.Warning(errors.New("Failed to convert fill_time to time.Time in update"))
  }
  return &fill_time_t
}

func (p *ParsedMessage) getFilledAvgPrice() *float64 {
  filled_avg_price := p.Get("data").Get("order").GetStringBytes("filled_avg_price")
  if filled_avg_price == nil {
    return nil
  }
  filled_avg_price_float, err := strconv.ParseFloat(string(filled_avg_price), 8)
  if err != nil {
    handlelog.Warning(errors.New("Failed to convert filled_avg_price to float in order update"))
  }
  return &filled_avg_price_float
}

func (a *Account) updateParser(p *ParsedMessage) *OrderUpdate {
  event := p.getEvent()
  if event == nil {
    return nil
  }
  strat_name := p.getStratName()
  if strat_name == nil {
    return nil
  }
  asset_class := p.getAssetClass()
  symbol := p.getSymbol()
  side := p.getSide()
  asset_qty := p.getAssetQty()
  fill_time := p.getFillTime()
  filled_avg_price := p.getFilledAvgPrice()

  return &OrderUpdate {
    Event:            event,
    AssetClass:       asset_class,
    StratName:        strat_name,
    Side:             side,
    Symbol:           symbol,
    AssetQty:         asset_qty,
    FillTime:         fill_time,
    FilledAvgPrice:   filled_avg_price,
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

func (a *Account) orderUpdateHandler(u *OrderUpdate) {
  var asset = a.assets[*u.AssetClass][*u.Symbol]
  var pos *Position = asset.Positions[*u.StratName]
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
    if *u.Event == "fill" || *u.Event == "canceled" {
      if pos.Qty.IsZero() {
        asset.RemovePosition(*u.StratName)
      } else {
        a.db_chan <-pos.LogOpen(*u.StratName)
        pos.OpenOrderPending = false
      }
    }
  // Close order logic
  } else if pos.CloseOrderPending {
    if u.FilledAvgPrice != nil {
      pos.CloseFilledAvgPrice = *u.FilledAvgPrice
    }
    if u.FillTime != nil {
      pos.CloseFillTime = *u.FillTime
    }
    if *u.Event == "fill" || *u.Event == "canceled" {
      a.db_chan <-pos.LogClose(*u.StratName)
      if pos.Qty.IsZero() {
        asset.RemovePosition(*u.StratName)
      } else {
        pos.CloseOrderPending = false
        asset.Close("IOC", *u.StratName)
      }
    }
  }
}
