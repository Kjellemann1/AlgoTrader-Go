
// Each Market instance is connected to a websocket for an exclusive asset_class. It listens for price updates from the
// market, as well as order updates from the Account instance, and updates the algo with this information.


  // log.Println("[ OK ]     ", msg)
  // log.Println("[ ERROR ]  ", )
  // log.Println("[ WARNING ]", )


package src

import (
  "sync"
  "log"
  "errors"
  "fmt"
  "slices"
  "strings"
  "github.com/valyala/fastjson"
  "github.com/gorilla/websocket"
  _"github.com/shopspring/decimal"

  "github.com/Kjellemann1/AlgoTrader-Go/src/constant"
  "github.com/Kjellemann1/AlgoTrader-Go/src/util/push"
  _"github.com/Kjellemann1/AlgoTrader-Go/src/util/prettyjson"
)


type Market struct {
  asset_class string
  db_chan chan string
  assets map[string]*Asset
  order_update_chan chan OrderUpdate
  conn *websocket.Conn
  url string
  parser fastjson.Parser
}


func (m *Market) onInitialMessages(element *fastjson.Value) {
  msg := string(element.GetStringBytes("msg"))
  switch msg {
    case "connected":
      log.Println("[ OK ]\tConnected to websocket for", m.asset_class)
    case "authenticated":
      log.Println("[ OK ]\tAuthenticated with websocket for", m.asset_class)
    default: // Check that all symbols are present in subscription
      var symbol_list_ptr *[]string
      if m.asset_class == "stock" {
        symbol_list_ptr = &constant.STOCK_LIST
      } else if m.asset_class == "crypto" {
        symbol_list_ptr = &constant.CRYPTO_LIST
      }
      // Make a map with two arrays with subscribtion symbols, one for bar and one for trades
      var subs = make(map[string][]string)
      for _, sub_type := range []string{"bars", "trades"} {
        // Get the array of symbols for the subscription type
        fj_array := element.GetArray(sub_type)
        subs[sub_type] = make([]string, len(fj_array))
        // Fill the array with the symbols
        for i, fj_symbol := range fj_array {
          subs[sub_type][i] = string(fj_symbol.GetStringBytes())
        }
        // Check that all symbols are present in the subscription
        for _, symbol := range subs[sub_type] {
          if slices.Contains(*symbol_list_ptr, symbol) == false {
            log.Panicln("Missing")
          }
        }
      }
      log.Printf("[ OK ]\tAll symbols present in websocket subscription for %s", m.asset_class)
  }
}

func (m *Market) CheckForSignal(symbol string) {
  s := Strategy{Asset: m.assets[symbol]}
  s.CheckForSignal()
}


func (m *Market) onMarketBarUpdate(element *fastjson.Value) {
  // TODO: Check within opening hours if stock
  symbol := string(element.GetStringBytes("S"))
  // TODO: Assert that the asset exists on initialization of the Market instance
  // and instead here check if the asset exists in the list of assets
  _, ok := m.assets[symbol]
  if !ok {
    // TODO: Handle this error
    return
  }
  m.assets[symbol].UpdateWindowOnBar(
    element.GetFloat64("o"),
    element.GetFloat64("h"),
    element.GetFloat64("l"),
    element.GetFloat64("c"),
    string(element.GetStringBytes("t")),
  )
  m.CheckForSignal(symbol)
}


func (m *Market) onMarketTradeUpdate(element *fastjson.Value) {
  // TODO: Check within opening hours if stock
  symbol := string(element.GetStringBytes("S"))
  t := string(element.GetStringBytes("t"))
  price := element.GetFloat64("p")
  fmt.Printf("Trade update: %s %s %f\n", symbol, t, price)
  if asset, ok := m.assets[symbol]; !ok {
    asset.UpdateWindowOnTrade(price, t)
  } else {
    // TODO: Handle this error
  }
}


func (m *Market) messageHandler(message []byte) error {
  arr, err := m.parser.ParseBytes(message)
  if err != nil {
    push.Error("Error parsing message in market.messageHandler(): ", err)
    log.Printf("Error message in market.messageHandler(): %s\n  -> message: %s\n", err, string(message))
    return errors.New("Error parsing message")
  }
  // Make sure arr is of type array as the API should return a list of messages
  if arr.Type() != fastjson.TypeArray {
    log.Printf("Message is not an array. Type: %s\n  -> Message %s\n", arr.Type(), message)
    return errors.New("Message is not an array")
  }
  // Handle each message based on the "T" field
  for _, element := range arr.GetArray() {
    message_type := string(element.GetStringBytes("T"))
    switch message_type {
      case "success", "subscription":
        m.onInitialMessages(element)
      case "b":
        m.onMarketBarUpdate(element)
      case "t":
        m.onMarketTradeUpdate(element)
      case "error":
        push.Error("Error message from websocket for ", errors.New(string(message)))
        log.Panicf("[ ERROR ]\tError message from websocket for %s\n  -> %s\n", m.asset_class, string(message))
      default:
        push.Warning("Unknown message type: ", errors.New(message_type))
        log.Printf("[ WARNING ]\tUnknown message type: %s\n  -> %s\n", message_type, string(message))
    }
  }
  return nil
}


func (m *Market) connect() {
  // Connect to the websocket
  conn, _, err := websocket.DefaultDialer.Dial(m.url, constant.AUTH_HEADERS)
  if err != nil {
    panic(err)
  }
  m.conn = conn
  // Receive connection and authentication messages
  for i := 0; i < 2; i++ {
    _, message, err := m.conn.ReadMessage()
    if err != nil {
      log.Panicln(err)
    }
    m.messageHandler(message)
  }
  // Subscbribe to symbols
  var sub_msg_symbols string = ""
  if m.asset_class == "stock" {
    sub_msg_symbols = strings.Join(constant.STOCK_LIST, "\",\"")
  }
  if m.asset_class == "crypto" {
    sub_msg_symbols = strings.Join(constant.CRYPTO_LIST, "\",\"")
  }
  sub_msg := []byte(fmt.Sprintf(`{"action":"subscribe", "trades":["%s"], "bars":["%s"]}`, sub_msg_symbols, sub_msg_symbols))
  if err := m.conn.WriteMessage(websocket.TextMessage, sub_msg); err != nil {
    panic(err)
  }
  // Receive subscription message
  _, sub_msg, err = m.conn.ReadMessage()
  if err != nil {
    log.Panicln(err)
  }
  m.messageHandler(sub_msg)
}


// Listens for market updates from the websocket connection
func (m *Market) marketUpdateListen(wg *sync.WaitGroup) {
  defer wg.Done()
  for {
    _, message, err := m.conn.ReadMessage()
    if err != nil {
      push.Error("Error reading message: ", err)
      log.Println("Error reading message: ", err)
    }
    if err := m.messageHandler(message); err != nil {
      push.Error("Error handling message: ", err)
      log.Println("Error handling message: ", err)
    }
  }
}


// Listen for order updates form the Account instance
func (m *Market) orderUpdateListen(wg *sync.WaitGroup) {
  defer wg.Done()
  for {
    // order_update := <-m.order_update_chan
  }
}


// Main listen function
func (m *Market) listen() error {
  wg := sync.WaitGroup{}
  // Listen to price updates from the market
  wg.Add(1)
  go m.marketUpdateListen(&wg)
  // Listen for order updates from Account
  wg.Add(1)
  go m.orderUpdateListen(&wg)
  wg.Wait()
  return nil
}


// Constructor
func NewMarket(asset_class string, url string, assets map[string]*Asset, db_chan chan string, order_update_chan chan OrderUpdate, wg *sync.WaitGroup) {
  defer wg.Done()
  // Check that all symbols are in the assets map
  var symbol_list_ptr *[]string
  if asset_class == "stock" {
    symbol_list_ptr = &constant.STOCK_LIST
  } else if asset_class == "crypto" {
    symbol_list_ptr = &constant.CRYPTO_LIST
  }
  for _, symbol := range *symbol_list_ptr {
    if _, ok := assets[symbol]; !ok {
      log.Panicln("Symbol not found in assets map: ", symbol)
    }
  }
  // Initialize Market instance
  m := Market{
    asset_class: asset_class,
    assets: assets,
    db_chan: db_chan,
    order_update_chan: order_update_chan,
    url: url,
  }
  m.connect()
  m.listen()
}
