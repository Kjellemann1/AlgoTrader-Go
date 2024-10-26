
/*
  Each Market instance is connected to a websocket for an exclusive asset_class. It listens for price updates from the
  market, and updates the symbols with this information. It uses a worker pool with the same size as the number of symbols
  to handle the updates concurrently. This is especially useful on bar updates, as the market can send updates for all symbols
  at the same time.
*/

package src

import (
  "sync"
  "log"
  "errors"
  "fmt"
  "slices"
  "time"
  "strings"
  "github.com/valyala/fastjson"
  "github.com/gorilla/websocket"

  "github.com/Kjellemann1/AlgoTrader-Go/src/constant"
  "github.com/Kjellemann1/AlgoTrader-Go/src/util/push"
)


func (m *Market) initiateWorkerPool(n_workers int, wg *sync.WaitGroup) {
  for i := 0; i < n_workers; i++ {
    wg.Add(1)
    go func() {
      defer wg.Done()
      for message := range m.worker_pool_chan {
        if err :=m.messageHandler(message); err != nil {
          push.Error("Error reading message: ", err)
          log.Println("Error reading message: ", err)
          continue
        }
      }
    }()
  }
}


type Market struct {
  asset_class       string
  assets            map[string]*Asset
  conn              *websocket.Conn
  url               string
  worker_pool_chan  chan []byte
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


func (m *Market) onMarketBarUpdate(element *fastjson.Value) {
  // TODO: Check within opening hours if stock
  symbol := string(element.GetStringBytes("S"))
  asset := m.assets[symbol]
  t, _ := time.Parse(time.RFC3339, string(element.GetStringBytes("t")))
  t = t.Add(1 * time.Minute)
  asset.UpdateWindowOnBar(
      element.GetFloat64("o"),
      element.GetFloat64("h"),
      element.GetFloat64("l"),
      element.GetFloat64("c"),
      t,
    )
  asset.CheckForSignal()
}


func (m *Market) onMarketTradeUpdate(element *fastjson.Value) {
  // TODO: Check within opening hours if stock
  symbol := string(element.GetStringBytes("S"))
  t, _ := time.Parse(time.RFC3339, string(element.GetStringBytes("t")))
  price := element.GetFloat64("p")
  asset := m.assets[symbol]
  asset.UpdateWindowOnTrade(price, t)
  asset.CheckForSignal()
}


func (m *Market) messageHandler(message []byte) error {
  parser := fastjson.Parser{}
  arr, err := parser.ParseBytes(message)
  if err != nil {
    push.Error("Error parsing message in market.messageHandler(): ", err)
    log.Printf(
      "Error message in market.messageHandler(): %s\n" + 
      "  -> message: %s\n",
    err, string(message))
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
      case "b":
        m.onMarketBarUpdate(element)
      case "t":
        m.onMarketTradeUpdate(element)
      case "success", "subscription":
        m.onInitialMessages(element)
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
  // Subscribe to symbols
  var sub_msg_symbols string = ""
  if m.asset_class == "stock" {
    sub_msg_symbols = strings.Join(constant.STOCK_LIST, "\",\"")
  }
  if m.asset_class == "crypto" {
    sub_msg_symbols = strings.Join(constant.CRYPTO_LIST, "\",\"")
  }
  sub_msg := []byte(fmt.Sprintf(`{"action":"subscribe", "trades":["%s"], "bars":["%s"]}`, sub_msg_symbols, sub_msg_symbols))
  if err := m.conn.WriteMessage(websocket.TextMessage, sub_msg); err != nil {
    log.Panicln(err.Error())
  }
  // Receive subscription message
  _, sub_msg, err = m.conn.ReadMessage()
  if err != nil {
    log.Panicln(err)
  }
  m.messageHandler(sub_msg)
}


// Listens for market updates from the websocket connection
func (m *Market) listen(n_workers int) {
  wg := sync.WaitGroup{}
  defer wg.Wait()

  m.initiateWorkerPool(n_workers, &wg)

  for {
    _, message, err := m.conn.ReadMessage()
    if err != nil {
      push.Error("Error reading message: ", err)
      log.Println("Error reading message: ", err)
      continue
    }
    m.worker_pool_chan <- message
  }
}


// Constructor
func NewMarket(asset_class string, url string, assets map[string]*Asset, wg *sync.WaitGroup) {
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
      log.Fatal("Symbol not found in assets map: ", symbol)
    }
  }
  // Initialize Market instance
  m := &Market{
    asset_class: asset_class,
    url: url,
    assets: assets,
    worker_pool_chan: make(chan []byte, len(*symbol_list_ptr)),
  }
  m.connect()
  m.listen(len(*symbol_list_ptr))
}
