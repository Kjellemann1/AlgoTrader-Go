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
  "github.com/Kjellemann1/AlgoTrader-Go/src/util/backoff"
  "github.com/Kjellemann1/AlgoTrader-Go/src/util/handlelog"
  "github.com/Kjellemann1/AlgoTrader-Go/src/order"
)

type MarketMessage struct {
  message []byte
  received_time time.Time
}

type Market struct {
  asset_class       string
  assets            map[string]*Asset
  conn              *websocket.Conn
  url               string
  worker_pool_chan  chan MarketMessage
}

func NewMarket(asset_class string, url string, assets map[string]*Asset) (m *Market) {
  // Check that all symbols are in the assets map
  var symbol_list_ptr *[]string
  if asset_class == "stock" {
    symbol_list_ptr = &constant.STOCK_LIST
  } else if asset_class == "crypto" {
    symbol_list_ptr = &constant.CRYPTO_LIST
  }
  for _, symbol := range *symbol_list_ptr {
    if _, ok := assets[symbol]; !ok {
      handlelog.ErrorPanic(errors.New("Symbol not found in assets map"), "Symbol", symbol)
    }
  }
  m = &Market{
    asset_class: asset_class,
    url: url,
    assets: assets,
    worker_pool_chan: make(chan MarketMessage, len(*symbol_list_ptr)),
  }
  return
}

func (m *Market) initiateWorkerPool(n_workers int, wg *sync.WaitGroup) {
  for i := 0; i < n_workers; i++ {
    wg.Add(1)
    go func() {
      defer wg.Done()
      for mm := range m.worker_pool_chan {
        if err :=m.messageHandler(mm); err != nil {
          handlelog.Warning(err, "Message", mm.message)
          continue
        }
      }
    }()
  }
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

func (m *Market) onMarketBarUpdate(element *fastjson.Value, received_time time.Time) {
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
      received_time,
    )
  asset.CheckForSignal()
}

func (m *Market) onMarketTradeUpdate(element *fastjson.Value, received_time time.Time) {
  // TODO: Check within opening hours if stock
  symbol := string(element.GetStringBytes("S"))
  t, _ := time.Parse(time.RFC3339, string(element.GetStringBytes("t")))
  price := element.GetFloat64("p")
  asset := m.assets[symbol]
  asset.UpdateWindowOnTrade(price, t, received_time)
  asset.CheckForSignal()
}

func (m *Market) messageHandler(mm MarketMessage) error {
  parser := fastjson.Parser{}
  arr, err := parser.ParseBytes(mm.message)
  if err != nil {
    handlelog.Warning(err, "Message", string(mm.message))
    return errors.New("Error parsing message")
  }
  // Make sure arr is of type array as the API should return a list of messages
  if arr.Type() != fastjson.TypeArray {
    err := errors.New("Message is not an array")
    handlelog.Warning(err, "Message", string(mm.message))
    return err
  }
  // Handle each message based on the "T" field
  for _, element := range arr.GetArray() {
    message_type := string(element.GetStringBytes("T"))
    switch message_type {
      case "b":
        m.onMarketBarUpdate(element, mm.received_time)
      case "t":
        m.onMarketTradeUpdate(element, mm.received_time)
      case "success", "subscription":
        m.onInitialMessages(element)
      case "error":
        err := errors.New(string(mm.message))
        handlelog.ErrorPanic(err, "Message", string(mm.message))
      default:
        handlelog.Warning(errors.New("Unknown message type"), "Message type", message_type, "Message", string(mm.message))
    }
  }
  return nil
}

func (m *Market) connect(initial *bool) error {
  // Connect to the websocket
  conn, _, err := websocket.DefaultDialer.Dial(m.url, constant.AUTH_HEADERS)
  if err != nil && *initial {
    log.Panic(err)
  } else if err != nil {
    return err
  }
  m.conn = conn
  // Receive connection and authentication messages
  for i := 0; i < 2; i++ {
    _, message, err := m.conn.ReadMessage()
    if err != nil && *initial {
      log.Panic(err)
    } else if err != nil {
      return err
    }
    m.messageHandler(MarketMessage{message, time.Now().UTC()})
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
    log.Panicln(err.Error())  // TODO: Cant have panic here on reconnect
  }
  // Receive subscription message
  _, sub_msg, err = m.conn.ReadMessage()
  if err != nil && *initial {
    log.Panic(err)
  } else if err != nil {
    return err
  }
  m.messageHandler(MarketMessage{sub_msg, time.Now().UTC()})
  *initial = false
  return nil
}

func (m *Market) PingPong() {
  if err := m.conn.SetReadDeadline(time.Now().Add(60 * time.Second)); err != nil {
    handlelog.Warning(err)
  }
  m.conn.SetPongHandler(func(string) error {
    m.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
    return nil
  })
  go func() {
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()
    for {
      select {
      case <-ticker.C:
        if err := m.conn.WriteControl(websocket.PingMessage, []byte("ping"), time.Now().Add(5 * time.Second)); err != nil {
          handlelog.Warning(err)
          m.conn.Close()
          return
        } else {
        }
      }
    }
  }()
}

func (m *Market) listen() error {
  defer m.conn.Close()
  wg := sync.WaitGroup{}
  defer wg.Wait()
  n_workers := len(m.assets)
  m.initiateWorkerPool(n_workers, &wg)
  defer close(m.worker_pool_chan)
  for {
    _, message, err := m.conn.ReadMessage()
    received_time := time.Now().UTC()
    if err != nil {
      if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway) {
        handlelog.Warning(err)
        return err
      } else if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
        handlelog.Warning(err)
        return err
      } else {
        handlelog.Warning(err, "Message", string(message))
        continue
      }
    }
    m.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
    m.worker_pool_chan <- MarketMessage{message, received_time}
  }
}

func (m *Market) Start(wg *sync.WaitGroup) {
  defer wg.Done()
  backoff_sec := 5
  retries := 0
  initial := true
  for {
    err := m.connect(&initial)
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
    m.PingPong()
    m.listen()
    err = m.conn.Close()
    if err != nil {
      handlelog.Error(errors.New("Could not close connection (after a.listen() call)"))
    }
  }
}
