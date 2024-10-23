
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
  "github.com/Kjellemann1/AlgoTrader-Go/src/util/push"
  "github.com/Kjellemann1/AlgoTrader-Go/src/constant"
  "github.com/Kjellemann1/AlgoTrader-Go/src/order"
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


func grepAction(orderID string) (string, error) {
  pattern := `action\[(.*?)\]`
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
  Action         string
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


func (a *Account) connect() {
  conn, response, err := websocket.DefaultDialer.Dial("wss://paper-api.alpaca.markets/stream", constant.AUTH_HEADERS)
  if err != nil {
    log.Panicf("[ ERROR ]\tCould not connect to account websocket: %s\n  -> %+v", err, response)
  }
  if err := conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf(`{"action":"auth","key":"%s","secret":"%s"}`, constant.KEY, constant.SECRET))); err != nil {
    log.Panicf("[ ERROR ]\tSending connection message to account websocket failed: %s", err)
  }
  _, message, err := conn.ReadMessage()
  if err != nil {
    log.Panicf("[ ERROR ]\tReading connection message from account websocket failed: %s\n  -> %s", err, message)
  }
  a.messageHandler(message)
  if err := conn.WriteMessage(websocket.TextMessage, []byte(`{"action":"listen","data":{"streams":["trade_updates"]}}`)); err != nil {
    log.Panicf("[ ERROR ]\tSending listen message to account websocket failed: %s", err)
  }
  a.conn = conn
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
    /*
      TODO: Might want to spawn a new goroutine for each order update. But needs
      each routine needs to be unique to the position, as the chronology of the
      order updates are important.

      However, handling order updates might not be important for performance. 
      What is important is the process from receiving the price to sending the order. 
      What comes after that, unless trading at a really high frequency, is not really 
      important as long as it doesnt block the other stuff. That has its own go routine. 
      And spawnind more goroutines for these order updates might actually steal resources
      from the more important stuff.
    */
    _, message, err := a.conn.ReadMessage()
    if err != nil {
      // TODO: Better error handling
      log.Println("Error reading message: ", err)
      panic(err)
    }
    a.messageHandler(message)
  }
}


func NewAccount(wg *sync.WaitGroup, assets map[string]map[string]*Asset, db_chan chan *Query) {
  defer wg.Done()
  a := &Account{
    parser: fastjson.Parser{},
    assets: assets,
    db_chan: db_chan,
  }
  a.connect()
  a.listen()
}


func calculatePositionQty(p *Position, a *Asset, u *OrderUpdate) {
  var position_change decimal.Decimal = (*u.AssetQty).Sub(a.AssetQty)
  p.Qty = p.Qty.Add(position_change)
  a.AssetQty = *u.AssetQty
  if !a.SumPosQtyEqAssetQty() {
    push.Error("Sum of position qty not equal to asset qty", nil)
    log.Printf(
      "[ FATAL ]\tSum of position qty not equal to asset qty\n" +
      "  -> Asset %d != %d Position\n" +
      "  -> OrderUpdate: %v+\n" +
      "  -> Closing all positions and shutting down\n",
      a.AssetQty, p.Qty, u,
    )
    order.CloseAllPositions(2, 0)
    log.Fatal("Shutting down")
  }
}


// TODO: Why on earth am I doing this here? This should happen in the Account instance.
// Then it wouldn't be necessary to send the order update to the Market instance.
func (a *Account) orderUpdateHandler(u *OrderUpdate) {
  var asset = a.assets[u.AssetClass][*u.Symbol]
  var pos *Position = asset.Positions[u.StratName]
  asset.mutex.Lock()
  defer asset.mutex.Unlock()
  // Update AssetQty
  if u.AssetQty != nil {
    calculatePositionQty(pos, asset, u)
  }
  if pos == nil {
    log.Panicf("Position nil: %s", *u.Symbol)
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
      pos.OpenOrderPending = false
      if pos.Qty.IsZero() {
        asset.RemovePosition(u.StratName)
      } else {
        a.db_chan <-pos.LogOpen(u.StratName)
      }
    }
  }
  // Close order logic
  if pos.CloseOrderPending {
    if u.FilledAvgPrice != nil {
      pos.CloseFilledAvgPrice = *u.FilledAvgPrice
    }
    if u.FillTime != nil {
      pos.CloseFillTime = *u.FillTime
    }
    if u.Event == "fill" || u.Event == "canceled" {
      pos.CloseOrderPending = false
      a.db_chan <-pos.LogClose(u.StratName)
      if pos.Qty.IsZero() {
        asset.RemovePosition(u.StratName)
      }
    }
  }
}


func (a *Account) updateParser(parsed_msg *fastjson.Value) *OrderUpdate {
  // Extract event. Shutdown if nil
  event := parsed_msg.Get("data").GetStringBytes("event")
  if event == nil {
    push.Error("EVENT NOT IN TRADE UPDATE\nSHUTTING DOWN\n", nil)
    log.Printf(
      "[ FATAL ]\tEvent not in trade update\n"+
      "  -> Closing all positions and shutting down\n" +
      "  -> Parsed message: %s\n",
    parsed_msg)
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
    push.Error("ASSET CLASS NOT IN TRADE UPDATE\nSHUTTING DOWN\n", nil)
    log.Printf(
      "[ FATAL ]\tAsset class not in trade update\n"+
      "  -> Closing all positions and shutting down\n" +
      "  -> Parsed message: %s\n",
    parsed_msg)
    order.CloseAllPositions(2, 0)
    log.Panicln("SHUTTING DOWN")
  }
  asset_class_str := string(asset_class)
  // Extract symbol, Return if nil
  var symbol_ptr *string
  symbol := parsed_msg.Get("data").Get("order").GetStringBytes("symbol")
  if symbol == nil {
    push.Error("SYMBOL NOT IN TRADE UPDATE\nSHUTTING DOWN\n", nil)
    log.Printf(
      "[ FATAL ]\tSymbol not in trade update\n"+
      "  -> Closing all positions and shutting down\n" +
      "  -> Parsed message: %s\n",
    parsed_msg)
    order.CloseAllPositions(2, 0)
    log.Panicln("SHUTTING DOWN")
  }
  symbol_str := string(symbol)
  symbol_ptr = &symbol_str
  // Extract PositionID. Return if nil
  order_id := parsed_msg.Get("data").Get("order").GetStringBytes("client_order_id")  // client_order_id == PositionID
  if order_id == nil {
    push.Warning("Order id not found", nil)
    log.Println("[ WARNING ]\tOrder ID not found")
    return nil
  }
  order_id_str := string(order_id)
  // Grep strat_name from order id. Return if nil
  strat_name, err := grepStratName(order_id_str)
  if err != nil {
    return nil
  }
  // Grep action from order_id. Return if nil
  action, err := grepAction(order_id_str)
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
      push.Error("FAILED TO CONVERT ASSET QTY TO decimal.Decimal", err)
      log.Printf(
        "[ FATAL ]\tFailed to convert asset qty to decimal.Decimal\n"+
        "  -> Closing all positions and shutting down\n"+
        "  -> Asset qty: %s\n" +
        "  -> Error: %s\n",
      asset_qty, err)
      order.CloseAllPositions(2, 0)
      log.Panicln("Shutting down")
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

  // Send update to Market instance
  return &OrderUpdate {
    Event:            event_str,
    AssetClass:       asset_class_str,
    StratName:        strat_name,
    Action:           action,
    Side:             side_ptr,
    Symbol:           symbol_ptr,
    AssetQty:         asset_qty_ptr,
    FillTime:         fill_time_ptr,
    FilledAvgPrice:   filled_avg_price_ptr,
  }
}
