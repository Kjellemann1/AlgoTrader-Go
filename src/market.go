package main

import (
  "sync"
  "log"
  "errors"
  "fmt"
  "slices"
  "time"
  "strings"
  "context"
  "github.com/valyala/fastjson"
  "github.com/gorilla/websocket"

  "github.com/Kjellemann1/AlgoTrader-Go/constant"
  "github.com/Kjellemann1/AlgoTrader-Go/util"
  "github.com/Kjellemann1/AlgoTrader-Go/order"
)

type MarketMessage struct {
  message       []byte
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
  n_workers := len(assets)
  m = &Market{
    asset_class: asset_class,
    url: url,
    assets: assets,
    worker_pool_chan: make(chan MarketMessage, n_workers),
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
          util.Warning(err, "Message", mm.message)
          continue
        }
      }
    }()
  }
}

func (m *Market) checkAllSymbolsInSubscription(element *fastjson.Value) {
  var symbol_list_ptr *[]string
  if m.asset_class == "stock" {
    symbol_list_ptr = &constant.STOCK_LIST
  } else if m.asset_class == "crypto" {
    symbol_list_ptr = &constant.CRYPTO_LIST
  }
  var subs = make(map[string][]string)
  for _, sub_type := range []string{"bars", "trades"} {
    fj_array := element.GetArray(sub_type)
    subs[sub_type] = make([]string, len(fj_array))
    for i, fj_symbol := range fj_array {
      subs[sub_type][i] = string(fj_symbol.GetStringBytes())
    }
    for _, symbol := range subs[sub_type] {
      if slices.Contains(*symbol_list_ptr, symbol) == false {
        log.Panicln("Missing symbol in subscription", symbol)
      }
    }
  }
  log.Printf("[ OK ]\tAll symbols present in websocket subscription for %s", m.asset_class)
}

func (m *Market) onInitialMessages(element *fastjson.Value) {
  msg := string(element.GetStringBytes("msg"))
  switch msg {
    case "connected":
      log.Println("[ OK ]\tConnected to websocket for", m.asset_class)
    case "authenticated":
      log.Println("[ OK ]\tAuthenticated with websocket for", m.asset_class)
    case "subscription":
      m.checkAllSymbolsInSubscription(element)
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

func (m *Market) parseMessage(mm MarketMessage) (*fastjson.Value, error) {
  parser := fastjson.Parser{}
  arr, err := parser.ParseBytes(mm.message)
  if err != nil {
    util.Warning(err, "Message", string(mm.message))
    return nil, err
  }
  if arr.Type() != fastjson.TypeArray {
    err := errors.New("Message is not an array")
    util.Warning(err, "Message", string(mm.message))
    return nil, err
  }
  return arr, err
}

func (m *Market) messageHandler(mm MarketMessage) error {
  arr, err := m.parseMessage(mm)
  if err != nil {
    return nil
  }
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
        util.ErrorPanic(err, "Message", string(mm.message))
      default:
        util.Warning(errors.New("Unknown message type"), "Message type", message_type, "Message", string(mm.message))
    }
  }
  return nil
}

func (m *Market) connect() (err error) {
  m.conn, _, err = websocket.DefaultDialer.Dial(m.url, constant.AUTH_HEADERS)
  if err != nil {
    return
  }
  var message []byte
  for i := 0; i < 2; i++ {
    _, message, err = m.conn.ReadMessage()
    if err != nil {
      log.Println("Message", string(message))
      return
    }
    m.messageHandler(MarketMessage{message, time.Now().UTC()})
  }

  if err = m.subscribe(); err != nil {
    return
  }

  return
}

func (m *Market) subscribe() (err error) {
  var sub_msg_symbols string = ""
  if m.asset_class == "stock" {
    sub_msg_symbols = strings.Join(constant.STOCK_LIST, "\",\"")
  }
  if m.asset_class == "crypto" {
    sub_msg_symbols = strings.Join(constant.CRYPTO_LIST, "\",\"")
  }
  sub_msg := []byte(fmt.Sprintf(`{"action":"subscribe", "trades":["%s"], "bars":["%s"]}`, sub_msg_symbols, sub_msg_symbols))
  if err = m.conn.WriteMessage(websocket.TextMessage, sub_msg); err != nil {
    return
  }
  _, sub_msg, err = m.conn.ReadMessage()
  if err != nil {
    return
  }
  m.messageHandler(MarketMessage{sub_msg, time.Now().UTC()})
  return
}

func (m *Market) PingPong(ctx context.Context) {
  if err := m.conn.SetReadDeadline(time.Now().Add(constant.READ_DEADLINE_SEC)); err != nil {
    util.Warning(err)
  }

  m.conn.SetPongHandler(func(string) error {
    m.conn.SetReadDeadline(time.Now().Add(constant.READ_DEADLINE_SEC))
    return nil
  })

  ticker := time.NewTicker(constant.PING_INTERVAL_SEC)
  defer ticker.Stop()
  log.Println("[ OK ]\tPingPong initiated for market websocket: ", m.asset_class)

  for {
    select {
    case <-ctx.Done():
      return
    case <-ticker.C:
      if err := m.conn.WriteControl(
        websocket.PingMessage, []byte("ping"),
        time.Now().Add(5 * time.Second)); err != nil {
        util.Warning(err)
        m.conn.Close()
        return
      }
    }
  }
}

func (m *Market) listen(ctx context.Context) error {
  wg := sync.WaitGroup{}
  defer wg.Wait()
  n_workers := len(m.assets)
  m.initiateWorkerPool(n_workers, &wg)
  defer close(m.worker_pool_chan)
  for {
    _, message, err := m.conn.ReadMessage()
    received_time := time.Now().UTC()
    if err != nil {
      select {
      case <-ctx.Done():
        return ctx.Err()
      default:
        util.Error(err)
        continue
      }
    }
    m.worker_pool_chan <- MarketMessage{message, received_time}
  }
}

func (m *Market) Start(wg *sync.WaitGroup, ctx context.Context) {
  defer wg.Done()
  backoff_sec := 5
  retries := 0
  initial := true

  for {
    select {
    case <-ctx.Done():
      return
    default:
      if err := m.connect(); err != nil {
        if initial {
          panic(err.Error())
        }
        if retries < 5 {
          util.Error(err, "Retries", retries)
        } else {
          util.Error(err,
            "MAXIMUM NUMBER OF RETRIES REACHED", retries, 
            "CLOSING ALL POSITIONS AND SHUTTING DOWN", "...",
          )
          order.CloseAllPositions(2, 0)
          log.Panic("SHUTTING DOWN")
        }
        util.Backoff(&backoff_sec)
        retries++
        continue
      }

      if err := m.subscribe(); err != nil {
        if initial {
          panic(err.Error())
        }
        util.Error(err)
        util.Backoff(&backoff_sec)
        retries++
        continue
      }

      initial = false
      backoff_sec = 5
      retries = 0

      go m.listen(ctx)
      m.PingPong(ctx)
      m.conn.Close()
    }
  }
}
