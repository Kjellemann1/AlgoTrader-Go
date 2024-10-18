
package src

import (
  "fmt"
  "log"
  "regexp"
  "errors"
  "sync"
  _"time"
  "github.com/gorilla/websocket"
  "github.com/valyala/fastjson"
  "github.com/Kjellemann1/AlgoTrader-Go/src/util/push"
  "github.com/Kjellemann1/AlgoTrader-Go/src/constant"
)


func grepStrategyName(orderID string) (string, error) {
  pattern := `strat\[(.*?)\]`
  re := regexp.MustCompile(pattern)
  match := re.FindStringSubmatch(orderID)

  if match != nil && len(match) > 1 {
    return match[1], nil // Returnerer innholdet inne i klammene
  }
  return "", errors.New("") // Returnerer tom streng hvis ingen match finnes
}


// Used to send order updates received in Account instance to MarketSocket instanc(es)
// through the order_update channel. 
type OrderUpdate struct {
  event string
  asset_class string
  strategy_name string
  side string
  symbol string
  asset_qty int
  fill_time string
  filled_avg_price float64
}


// Account is the main struct for handling order updates
type Account struct {
  conn *websocket.Conn
  parser fastjson.Parser
  order_update_chan map[string] chan OrderUpdate
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
    log.Println("[ OK ]\tAuthorized with account websocket")
  } else {
    log.Panicln("[ ERROR ]\tAuthorization with account websocket failed")
  }
}


func (a *Account) onTradeUpdate(element *fastjson.Value) {
  event := string(element.Get("data").GetStringBytes("event"))
  // All possible events are listed here. https://docs.alpaca.markets/docs/websocket-streaming. Not all are not handled.
  if event != "fill" && event != "partial_fill" && event != "canceled" {
    return
  }
  asset_class := string(element.Get("data").Get("order").GetStringBytes("asset_class"))
  if asset_class == "us_equity" {
    asset_class = "stock"
  }
  fill_time := string(element.Get("data").Get("order").GetStringBytes("filled_at"))
  order_id := string(element.Get("data").Get("order").GetStringBytes("client_order_id"))
  strategy_name, err := grepStrategyName(order_id)
  if err != nil {
    push.Error("Grepping strategy name failed", err)
    log.Printf("[ Error ]\tGrepping strategy name failed\n  -> %s", order_id)
    return
  }
  filled_avg_price := element.Get("data").Get("order").GetFloat64("filled_avg_price")
  thread_message := OrderUpdate{
    event: event,
    strategy_name: strategy_name,
    side: string(element.Get("data").Get("order").GetStringBytes("side")),
    symbol: string(element.Get("data").Get("order").GetStringBytes("symbol")),
    asset_qty: element.Get("data").Get("position_qty").GetInt(),
    fill_time: fill_time,
    filled_avg_price: filled_avg_price,
  }
  a.order_update_chan[asset_class] <- thread_message
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
      a.onTradeUpdate(parsed_msg)
    case "listening":
      log.Println("[ OK ]\tListening to order updates")
      panic("END HERE")
  }
}


func (a *Account) listen() {
  for {
    _, message, err := a.conn.ReadMessage()
    if err != nil {
      fmt.Println("Error reading message: ", err)
      panic(err)
    }
    a.messageHandler(message)
  }
}


func NewAccount(order_update_chan map[string] chan OrderUpdate, wg *sync.WaitGroup) {
  defer wg.Done()
  a := &Account{
    parser: fastjson.Parser{},
    order_update_chan: order_update_chan,
  }
  a.connect()
  a.listen()
}
