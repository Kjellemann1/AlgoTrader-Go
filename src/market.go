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
  "github.com/Kjellemann1/AlgoTrader-Go/request"
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

  listen            func(*sync.WaitGroup, chan int8)
  pingPong          func(*sync.WaitGroup, context.Context, chan int8)
}

func NewMarket(asset_class string, url string, assets map[string]*Asset) (m *Market) {
  n_workers := len(assets)
  m = &Market{
    asset_class: asset_class,
    url: url,
    assets: assets,
    worker_pool_chan: make(chan MarketMessage, n_workers),
  }
  m.pingPong = m.pingPongFunc
  m.listen = m.listenFunc
  return
}

func (m *Market) initiateWorkerPool(n_workers int) {
  var wg sync.WaitGroup
  for range n_workers {
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
  symbols := []string{}
  for s := range m.assets {
    symbols = append(symbols, s)
  }

  var subs = make(map[string][]string)
  for _, sub_type := range []string{"bars", "trades"} {
    fj_array := element.GetArray(sub_type)
    subs[sub_type] = make([]string, len(fj_array))

    for i, fj_symbol := range fj_array {
      subs[sub_type][i] = string(fj_symbol.GetStringBytes())
    }

    for _, symbol := range subs[sub_type] {
      if !slices.Contains(symbols, symbol) {
        log.Panicln("Missing symbol in subscription", symbol)
      }
    }
  }

  util.Ok(fmt.Sprintf("All symbols present in websocket subscription for %s", m.asset_class))
}

func (m *Market) onInitialMessages(element *fastjson.Value) {
  msg := string(element.GetStringBytes("msg"))
  switch msg {
    case "connected":
      util.Ok(fmt.Sprintf("Connected to websocket for %s", m.asset_class))
    case "authenticated":
      util.Ok(fmt.Sprintf("Authenticated with websocket for %s", m.asset_class))
    default: // subscription
      m.checkAllSymbolsInSubscription(element)
  }
}

func (m *Market) onMarketBarUpdate(element *fastjson.Value, received_time time.Time) {
  // TODO: Check within opening hours if stock
  asset := m.assets[string(element.GetStringBytes("S"))]
  t, _ := time.Parse(time.RFC3339, string(element.GetStringBytes("t")))
  t = t.Add(1 * time.Minute)

  asset.updateWindowOnBar(
    element.GetFloat64("o"),
    element.GetFloat64("h"),
    element.GetFloat64("l"),
    element.GetFloat64("c"),
    t,
    received_time,
  )
  asset.checkForSignal()
}

func (m *Market) onMarketTradeUpdate(element *fastjson.Value, received_time time.Time) {
  // TODO: Check within opening hours if stock
  t, _ := time.Parse(time.RFC3339, string(element.GetStringBytes("t")))
  price := element.GetFloat64("p")
  asset := m.assets[string(element.GetStringBytes("S"))]
  asset.updateWindowOnTrade(price, t, received_time)
  asset.checkForSignal()
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
        return err
      default:
        util.Warning(errors.New("Unknown message type"), "Message type", message_type, "Message", string(mm.message))
    }
  }
  return nil
}

func (m *Market) connect() (err error) {
  m.conn, _, err = websocket.DefaultDialer.Dial(m.url, constant.AUTH_HEADERS)
  if err != nil {
    return err
  }
  var message []byte
  for range 2 {
    _, message, err = m.conn.ReadMessage()
    if err != nil {
      return err
    }
    err = m.messageHandler(MarketMessage{message, time.Now().UTC()})
    if err != nil {
      return err
    }
  }
  return nil
}

func (m *Market) subscribe() (err error) {
  symbols := []string{}
  for s := range m.assets {
    symbols = append(symbols, s)
  }
  sub_msg_symbols := strings.Join(symbols, "\",\"")
  sub_msg := fmt.Appendf(make([]byte, 0), `{"action":"subscribe", "trades":["%s"], "bars":["%s"]}`, 
    sub_msg_symbols, sub_msg_symbols, 
  )
  if err = m.conn.WriteMessage(websocket.TextMessage, sub_msg); err != nil {
    return
  }
  _, sub_msg, err = m.conn.ReadMessage()
  if err != nil {
    return
  }
  err = m.messageHandler(MarketMessage{sub_msg, time.Now().UTC()})
  if err != nil {
    return
  }
  return
}

func (m *Market) pingPongFunc(connWg *sync.WaitGroup, ctx context.Context, err_chan chan int8) {
  defer connWg.Done()

  if err := m.conn.SetReadDeadline(time.Now().Add(constant.READ_DEADLINE_SEC)); err != nil {
    util.Warning(err)
  }

  m.conn.SetPongHandler(func(string) error {
    err := m.conn.SetReadDeadline(time.Now().Add(constant.READ_DEADLINE_SEC))
    if err != nil {
      return err
    }
    return nil
  })

  ticker := time.NewTicker(constant.PING_INTERVAL_SEC)
  defer ticker.Stop()
  util.Ok(fmt.Sprintf("PingPong initiated for %s market websocket", m.asset_class))

  for {
    select {
    case <-ctx.Done():
      return
    case <-ticker.C:
      if err := m.conn.WriteControl(
        websocket.PingMessage, []byte("ping"),
        time.Now().Add(5 * time.Second)); err != nil {
        select {
        case err_chan <-1:
          util.Error(err)
        default:
        }
        return
      }
    }
  }
}

func (m *Market) listenFunc(connWg *sync.WaitGroup, err_chan chan int8) {
  defer connWg.Done()

  for {
    _, message, err := m.conn.ReadMessage()
    received_time := time.Now().UTC()
    if err != nil {
      select {
      case err_chan <-1:
        util.Error(err)
      default:
      }
      return
    }

    m.worker_pool_chan <- MarketMessage{message, received_time}
  }
}

func (m *Market) start(wg *sync.WaitGroup, ctx context.Context, backoff_sec_min float64) {
  defer wg.Done()

  defer close(m.worker_pool_chan)
  m.initiateWorkerPool(len(m.assets))

  backoff_sec := backoff_sec_min
  retries := 0

  for {
    if err := m.connect(); err != nil {
      if retries < 5 {
        util.Error(err, "Retries", retries)
        util.Backoff(&backoff_sec)
        retries++
        continue
      } else {
        util.Error(err, "Max retries reached", retries, "CLOSING ALL POSITIONS AND SHUTTING DOWN", "...")
        request.CloseAllPositions(2, 0)
        log.Panicln("SHUTTING DOWN")
      }
    }

    if err := m.subscribe(); err != nil {
      util.Error(err)
      util.Backoff(&backoff_sec)
      retries++
      continue
    }

    backoff_sec = backoff_sec_min
    retries = 0

    err_chan := make(chan int8)

    var connWg sync.WaitGroup
    connWg.Add(1)
    go m.listen(&connWg, err_chan)

    context, cancel := context.WithCancel(ctx)
    connWg.Add(1)
    go m.pingPong(&connWg, context, err_chan)

    select {
    case <-ctx.Done():
      cancel()
      m.conn.Close()
      connWg.Wait()
      return
    case <-err_chan:
      cancel()
      m.conn.Close()
      connWg.Wait()
    }
  }
}
