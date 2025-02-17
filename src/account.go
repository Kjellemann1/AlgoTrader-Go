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
  "github.com/gorilla/websocket"
  "github.com/valyala/fastjson"
  "github.com/shopspring/decimal"
  "github.com/Kjellemann1/AlgoTrader-Go/src/constant"
  "github.com/Kjellemann1/AlgoTrader-Go/src/order"
  "github.com/Kjellemann1/AlgoTrader-Go/src/util/handlelog"
  "github.com/Kjellemann1/AlgoTrader-Go/src/util/backoff"
  "github.com/Kjellemann1/AlgoTrader-Go/src/util/pretty"
)

func grepStratName(orderID string) (string, error) {
  pattern := `strat\[(.*?)\]`
  re := regexp.MustCompile(pattern)
  match := re.FindStringSubmatch(orderID)
  if match != nil && len(match) > 1 {
    return match[1], nil
  }
  return "", errors.New("")
}

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
      update := a.updateParser(parsed_msg)
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

func (a *Account) PingPong(ctx context.Context) {
  if err := a.conn.SetReadDeadline(time.Now().Add(60 * time.Second)); err != nil {
    handlelog.Warning(err)
  }
  a.conn.SetPongHandler(func(string) error {
    a.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
    return nil
  })
  go func() {
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
          a.conn.Close()
          return
        }
      }
    }
  }()
}

func (a *Account) ordersPending() bool {
  for _, class := range a.assets {
    for _, asset := range class {
      for _, position := range (*asset).Positions {
        if position.OpenOrderPending || position.CloseOrderPending {
          if position.OpenOrderPending {
            fmt.Println("Open order pending for:", position.Symbol)
          } else if position.CloseOrderPending {
            fmt.Println("Close order pending for:", position.Symbol)
          }
          return true
        }
      }
    }
  }
  return false
}

func (a *Account) listen(ctx context.Context) {
  for {
    _, message, err := a.conn.ReadMessage()
    if err != nil {
      select {
      case <-ctx.Done():
        return
      default:
        if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway) {
          handlelog.Warning(err)
          a.conn.Close()
          return
        } else if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
          handlelog.Warning(err)
          a.conn.Close()
          return
        } else {
          NNP.NoNewPositionsTrue("Account.listen")
          handlelog.Error(err, "Message", string(message), "NoNewPositions", NNP.Flag)
          continue
        }
      }
    }
    a.messageHandler(message)
  }
}

func (a *Account) Start(wg *sync.WaitGroup, ctx context.Context) {
  defer wg.Done()
  backoff_sec := 5
  retries := 0
  initial := true
  go func() {
    ticker := time.NewTicker(2 * time.Second)
    defer ticker.Stop()
    <-ctx.Done()
    for range ticker.C {
      if a.ordersPending() {
        log.Println("Order pending ...")
      } else {
        a.db_chan <- nil
        a.conn.Close()
        return
      }
    }
  }()
  for {
    select {
    case <-ctx.Done():
      return
    default:
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
    }
    backoff_sec = 5
    retries = 0
    a.PingPong(ctx)
    a.listen(ctx)
    // TODO: On reconnect, check that positions on the server are the same as locally
    //   -> Make sure we didn't miss any updates
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

func (a *Account) updateParser(parsed_msg *fastjson.Value) *OrderUpdate {
  fmt.Println("  -> Account Update:")
  pretty.PrintFormattedJSON(parsed_msg)
  // Extract event. Shutdown if nil
  event := parsed_msg.Get("data").GetStringBytes("event")
  if event == nil {
    handlelog.Error(
      errors.New("EVENT NOT IN TRADE UPDATE"), "Parsed message", parsed_msg,
      "CLOSING ALL POSITIONS AND SHUTTING DOWN", "...",
    )
    order.CloseAllPositions(2, 0)
    log.Panicln("SHUTTING DOWN")
  }
  event_str := string(event)
  // Only handle fill, partial_fill and canceled events.
  // Other events are likely not relevant. https://alpaca.markets/docs/api-documentation/api-v2/streaming/
  if event_str != "fill" && event_str != "partial_fill" && event_str != "canceled" {
    return nil
  }
  // Extract asset class. Shutdown if nil
  asset_class := parsed_msg.Get("data").Get("order").GetStringBytes("asset_class")
  if asset_class == nil {
    handlelog.Error(
      errors.New("ASSET CLASS NOT IN TRADE UPDATE"), "Parsed message", parsed_msg,
      "CLOSING ALL POSITIONS AND SHUTTING DOWN", "...",
    )
    order.CloseAllPositions(2, 0)
    log.Panicln("SHUTTING DOWN")
  }
  asset_class_str := string(asset_class)
  if asset_class_str == "us_equity" {
    asset_class_str = "stock"
  }
  // Extract symbol, Return if nil
  var symbol_ptr *string
  symbol := parsed_msg.Get("data").Get("order").GetStringBytes("symbol")
  if symbol == nil {
    handlelog.Error(
      errors.New("SYMBOL NOT IN TRADE UPDATE"), "Parsed message", parsed_msg,
      "CLOSING ALL POSITIONS AND SHUTTING DOWN", "...",
    )
    order.CloseAllPositions(2, 0)
    log.Panicln("SHUTTING DOWN")
  }
  symbol_str := string(symbol)
  symbol_ptr = &symbol_str
  // Extract PositionID. Return if nil
  order_id := parsed_msg.Get("data").Get("order").GetStringBytes("client_order_id")  // client_order_id == PositionID
  if order_id == nil {
    handlelog.Warning(errors.New("Order id not found"), nil)
    return nil
  }
  order_id_str := string(order_id)
  // Grep strat_name from order id. Return if nil
  strat_name, err := grepStratName(order_id_str)
  if err != nil {
    return nil
  }
  // Extract side
  var side_ptr *string
  side := parsed_msg.Get("data").Get("order").GetStringBytes("side")
  if side != nil {
    side_str := string(side)
    side_ptr = &side_str
  }
  // Extract asset_qty
  var asset_qty_ptr *decimal.Decimal
  asset_qty := parsed_msg.Get("data").GetStringBytes("position_qty")
  if asset_qty != nil {
    asset_qty_dec, err := decimal.NewFromString(string(asset_qty))
    if err != nil {
      handlelog.Error(err, "Asset qty", asset_qty, "CLOSING ALL POSITIONS AND SHUTTING DOWN", "...")
      order.CloseAllPositions(2, 0)
      log.Panicln("SHUTTING DOWN")
    }
    asset_qty_ptr = &asset_qty_dec
  }
  // Extract fill_time
  var fill_time_ptr *time.Time
  fill_time_byte := parsed_msg.Get("data").Get("order").GetStringBytes("filled_at")
  if fill_time_byte != nil {
    fill_time, _ := time.Parse(time.RFC3339, string(fill_time_byte))
    fill_time_ptr = &fill_time
  }
  // Extract filled_avg_price
  var filled_avg_price_ptr *float64
  filled_avg_price := parsed_msg.Get("data").Get("order").GetStringBytes("filled_avg_price")
  if filled_avg_price != nil {
    filled_avg_price_float, _ := strconv.ParseFloat(string(filled_avg_price), 8)
    filled_avg_price_ptr = &filled_avg_price_float
  }

  return &OrderUpdate {
    Event:            event_str,
    AssetClass:       asset_class_str,
    StratName:        strat_name,
    Side:             side_ptr,
    Symbol:           symbol_ptr,
    AssetQty:         asset_qty_ptr,
    FillTime:         fill_time_ptr,
    FilledAvgPrice:   filled_avg_price_ptr,
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
    if u.Event == "fill" || u.Event == "canceled" {
      a.db_chan <-pos.LogClose(u.StratName)
      if pos.Qty.IsZero() {
        asset.RemovePosition(u.StratName)
      } else {
        pos.CloseOrderPending = false
        asset.Close("IOC", u.StratName)
      }
    }
  }
}
