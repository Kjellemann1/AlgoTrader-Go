
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


func grepStratName(orderID string) (string, error) {
  pattern := `strat\[(.*?)\]`
  re := regexp.MustCompile(pattern)
  match := re.FindStringSubmatch(orderID)
  if match != nil && len(match) > 1 {
    return match[1], nil
  }
  return "", errors.New("")
}


// Used to send order updates received in Account instance to MarketSocket instanc(es)
// through the order_update channel. 
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


// Account is the main struct for handling order updates
type Account struct {
  conn *websocket.Conn
  parser fastjson.Parser
  db_chan chan *Query
  assets map[string]map[string]*Asset
  // TODO stock and crypto cleared for shutdown
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


func (a *Account) listen() {
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
    a.listen()
    backoff_sec = 5
    retries = 0
    // TODO: On reconnect, check that positions on the server are the same as locally
    a.conn.Close()
  }

}


// Constructor
func NewAccount(assets map[string]map[string]*Asset, db_chan chan *Query) *Account {
  a := &Account{
    parser: fastjson.Parser{},
    assets: assets,
    db_chan: db_chan,
  }
  return a
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
