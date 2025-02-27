package main

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
  "github.com/Kjellemann1/AlgoTrader-Go/constant"
  "github.com/Kjellemann1/AlgoTrader-Go/request"
  "github.com/Kjellemann1/AlgoTrader-Go/util"
)

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
  return &Account{
    parser: fastjson.Parser{},
    assets: assets,
    db_chan: db_chan,
  }
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
    util.Error(err)
    return
  }

  message_type := string(parsed_msg.GetStringBytes("stream"))
  switch message_type {
    case "authorization":
      a.onAuth(parsed_msg)
    case "trade_updates":
      update := a.updateParser(parsed_msg)
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

  err = a.conn.WriteMessage(websocket.TextMessage, 
    fmt.Appendf(make([]byte, 0), `{"action":"auth","key":"%s","secret":"%s"}`,
    constant.KEY, constant.SECRET),
  )
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

func (a *Account) PingPong(ctx context.Context, err_chan chan error) {
  if err := a.conn.SetReadDeadline(time.Now().Add(constant.READ_DEADLINE_SEC)); err != nil {
    util.Warning(err)
  }

  a.conn.SetPongHandler(func(string) error {
    err := a.conn.SetReadDeadline(time.Now().Add(constant.READ_DEADLINE_SEC))
    if err != nil {
      util.Warning(err)
    }
    return nil
  })

  ticker := time.NewTicker(constant.PING_INTERVAL_SEC)
  defer ticker.Stop()
  log.Println("[ OK ]\tPingPong initiated for account websocket")

  for {
    select {
    case <-ctx.Done():
      return
    case <-ticker.C:
      if err := a.conn.WriteControl(
        websocket.PingMessage, []byte("ping"), 
        time.Now().Add(5 * time.Second)); err != nil {
        err_chan <-err
        return
      }
    }
  }
}

func (a *Account) listen(ctx context.Context, err_chan chan error) {
  for {
    _, message, err := a.conn.ReadMessage()
    if err != nil {
      select {
      case <-ctx.Done():
        return
      default:
        err_chan <-err
        return
      }
    }
    a.messageHandler(message)
  }
}

func (a *Account) start(wg *sync.WaitGroup, ctx context.Context) {
  defer wg.Done()
  backoff_sec := 2
  retries := 0

  for {
    if err := a.connect() ; err != nil {
      if retries < 5 {
        util.Error(err, "Retries", retries)
      } else {
        NNP.NoNewPositionsTrue("")
        util.Error(err, 
          "MAXIMUM NUMBER OF RETRIES REACHED", retries, 
          "CLOSING ALL POSITIONS AND SHUTTING DOWN", "...",
        )
        request.CloseAllPositions(2, 0)
        log.Panic("SHUTTING DOWN")
      }

      util.Backoff(&backoff_sec)
      retries++
      continue
    }

    backoff_sec = 5
    retries = 0

    err_chan := make(chan error)
    childCtx, cancel := context.WithCancel(ctx)

    go a.listen(childCtx, err_chan)
    go a.PingPong(childCtx, err_chan)

    select {
    case <-ctx.Done():
      cancel()
      return
    case err := <-err_chan:
      util.Error(err)
      cancel()
      a.conn.Close()
    }
  }
}

func (a *Account) getEvent(data *fastjson.Value) *string {
  // Shutdown if nil
  // Only handle fill, partial_fill and canceled events.
  // Other events are likely not relevant. https://alpaca.markets/docs/api-documentation/api-v2/streaming/
  event := data.GetStringBytes("event")
  if event == nil {
    NNP.NoNewPositionsTrue("")
    util.Error(
      errors.New("EVENT NOT IN TRADE UPDATE"), "Parsed message", data.String(),
      "CLOSING ALL POSITIONS AND SHUTTING DOWN", "...",
    )
    request.CloseAllPositions(2, 0)
    log.Panicln("SHUTTING DOWN")
  }

  event_str := string(event)
  if event_str != "fill" && event_str != "partial_fill" && event_str != "canceled" {
    return nil
  }

  return &event_str
}

func (p *Account) getPositionID(order *fastjson.Value) *string {
  position_id := order.GetStringBytes("client_order_id")  // client_order_id == PositionID
  if position_id == nil {
    util.Warning(errors.New("PositionID not found in order update"), nil)
    return nil
  }

  position_id_str := string(position_id)

  return &position_id_str
}

func grepStratName(position_id *string) *string {
  pattern := `strat\[(.*?)\]`
  re := regexp.MustCompile(pattern)
  match := re.FindStringSubmatch(*position_id)
  if len(match) > 1 {
    return &match[1]
  }

  return nil
}

func (a *Account) getStratName(order *fastjson.Value) *string {
  // Return if nil 
  position_id := a.getPositionID(order)

  if position_id == nil {
    return nil
  }

  return grepStratName(position_id)
}

func (a *Account) getAssetClass(order *fastjson.Value) *string {
  // Shutdown if nil
  asset_class := order.GetStringBytes("asset_class")
  if asset_class == nil {
    NNP.NoNewPositionsTrue("")
    util.Error(
      errors.New("ASSET CLASS NOT IN TRADE UPDATE"), "Parsed message", order.String(),
      "CLOSING ALL POSITIONS AND SHUTTING DOWN", "...",
    )
    request.CloseAllPositions(2, 0)
    log.Panicln("SHUTTING DOWN")
  }

  asset_class_str := string(asset_class)
  if asset_class_str == "us_equity" {
    asset_class_str = "stock"
  }

  return &asset_class_str
}

func (a *Account) getSymbol(order *fastjson.Value) *string {
  // Return if nil, shutdown if fails
  symbol := order.GetStringBytes("symbol")
  if symbol == nil {
    NNP.NoNewPositionsTrue("")
    util.Error(
      errors.New("SYMBOL NOT IN TRADE UPDATE"), "Parsed message", order.String(),
      "CLOSING ALL POSITIONS AND SHUTTING DOWN", "...",
    )
    request.CloseAllPositions(2, 0)
    log.Panicln("SHUTTING DOWN")
  }

  symbol_str := string(symbol)

  return &symbol_str
}

func (a *Account) getSide(order *fastjson.Value) *string {
  side := order.GetStringBytes("side")
  if side == nil {
    return nil
  }

  side_str := string(side)

  return &side_str
}

func (a *Account) getAssetQty(data *fastjson.Value) *decimal.Decimal {
  // Return if nil, shutdown if fails
  asset_qty := data.GetStringBytes("position_qty")
  if asset_qty == nil {
    return nil
  }

  asset_qty_dec, err := decimal.NewFromString(string(asset_qty))
  if err != nil {
    NNP.NoNewPositionsTrue("")
    util.Error(err, "Asset qty", asset_qty, "CLOSING ALL POSITIONS AND SHUTTING DOWN", "...")
    request.CloseAllPositions(2, 0)
    log.Panicln("SHUTTING DOWN")
  }

  return &asset_qty_dec
}

func (a *Account) getFillTime(order *fastjson.Value) *time.Time {
  fill_time := order.GetStringBytes("filled_at")
  if fill_time == nil {
    return nil
  }

  fill_time_t, err := time.Parse(time.RFC3339, string(fill_time))
  if err != nil {
    util.Warning(errors.New("Failed to convert fill_time to time.Time"))
  }

  return &fill_time_t
}

func (a *Account) getFilledAvgPrice(order *fastjson.Value) *float64 {
  filled_avg_price := order.GetStringBytes("filled_avg_price")
  if filled_avg_price == nil {
    return nil
  }

  filled_avg_price_float, err := strconv.ParseFloat(string(filled_avg_price), 64)
  if err != nil {
    util.Warning(errors.New("Failed to convert filled_avg_price to float"))
  }

  return &filled_avg_price_float
}

func (a *Account) updateParser(pm *fastjson.Value) *OrderUpdate {
  data := pm.Get("data")
  order := data.Get("order")

  event := a.getEvent(data)
  if event == nil {
    return nil
  }

  strat_name := a.getStratName(order)
  if strat_name == nil {
    return nil
  }

  asset_qty := a.getAssetQty(data)
  asset_class := a.getAssetClass(order)
  symbol := a.getSymbol(order)
  side := a.getSide(order)
  fill_time := a.getFillTime(order)
  filled_avg_price := a.getFilledAvgPrice(order)

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

func updateAssetQty(p *Position, a *Asset, u *OrderUpdate) {
  a.Rwm.Lock()
  defer a.Rwm.Unlock()

  var position_change decimal.Decimal = (*u.AssetQty).Sub(a.Qty)
  p.Qty = p.Qty.Add(position_change)
  a.Qty = *u.AssetQty

  if !a.sumPosQtysEqAssetQty() {
    util.Error(
      errors.New("Sum of position qty not equal to asset qty"),
      "Asset", a.Qty, "Position", p.Qty, "OrderUpdate", u,
      "CLOSING ALL POSITIONS AND SHUTTING DOWN", "...",
    )
    request.CloseAllPositions(2, 0)
    log.Fatal("SHUTTING DOWN")
  }
}

func (a *Account) closeLogic(asset *Asset, pos *Position, u *OrderUpdate) {
  if u.FilledAvgPrice != nil {
    pos.CloseFilledAvgPrice = *u.FilledAvgPrice
  }

  if u.FillTime != nil {
    pos.CloseFillTime = *u.FillTime
  }

  if *u.Event == "fill" || *u.Event == "canceled" {
    a.db_chan <-pos.LogClose()
    if pos.Qty.IsZero() {
      asset.removePosition(*u.StratName)
    } else {
      asset.Mutex.Lock()
      pos.CloseOrderPending = false
      asset.close("IOC", *u.StratName)
      asset.Mutex.Unlock()
    }
  }
}

func (a *Account) openLogic(asset *Asset, pos *Position, u *OrderUpdate) {
  if u.FilledAvgPrice != nil {
    pos.OpenFilledAvgPrice = *u.FilledAvgPrice
  }

  if u.FillTime != nil {
    pos.OpenFillTime = *u.FillTime
  }

  if *u.Event == "fill" || *u.Event == "canceled" {
    if pos.Qty.IsZero() {
      asset.removePosition(*u.StratName)
    } else {
      a.db_chan <-pos.LogOpen()
      pos.OpenOrderPending = false
    }
  }
}

func (a *Account) reconnectDiff(u *OrderUpdate) {
  asset := a.assets[*u.AssetClass][*u.Symbol]
  asset.Rwm.Lock()
  defer asset.Rwm.Unlock()

  if !asset.sumPosQtysEqAssetQty() {
    NNP.NoNewPositionsTrue("")
    util.Error(errors.New("Position quantities do not sum to asset qty"),
      "CLOSING ALL POSITIONS AND SHUTTING DOWN", "...",
    )
    request.CloseAllPositions(2, 0)
    log.Panicln("SHUTTING DOWN")
  }
}

func (a *Account) orderUpdateHandler(u *OrderUpdate) {
  if *u.StratName == "reconnect_multiple_diff" {
    a.reconnectDiff(u)
    return
  }

  var asset = a.assets[*u.AssetClass][*u.Symbol]
  var pos *Position = asset.Positions[*u.StratName]

  if pos == nil {
    util.Error(errors.New("Position nil: %s"),
      "Symbol", *u.Symbol,
      "StratName", *u.StratName,
      "CLOSING ALL POSITIONS AND SHUTTING DOWN", "...",
    )
    request.CloseAllPositions(2, 0)
    log.Panicln("SHUTTING DOWN")
  }

  pos.Rwm.Lock()
  defer pos.Rwm.Unlock()

  if u.AssetQty != nil {
    updateAssetQty(pos, asset, u)
  }

  if pos.OpenOrderPending {
    a.openLogic(asset, pos, u)
  } else if pos.CloseOrderPending {
    a.closeLogic(asset, pos, u)
  }
}
