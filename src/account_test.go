package main

import (
  "io"
  "time"
  "errors"
  "testing"
  "net/http"
  "net/http/httptest"
  "strings"
  "sync"
  "context"
  "github.com/gorilla/websocket"
  "github.com/stretchr/testify/assert"
  "github.com/shopspring/decimal"
  "github.com/valyala/fastjson"
  "github.com/Kjellemann1/AlgoTrader-Go/request"
  "github.com/Kjellemann1/AlgoTrader-Go/push"
  "github.com/Kjellemann1/AlgoTrader-Go/constant"
  "github.com/Kjellemann1/AlgoTrader-Go/util"
)

type mockAccountConn struct {
  *websocket.Conn
}

func (c *mockAccountConn) write(data string) {
  msg := []byte(data)
  _ = c.WriteMessage(1, msg)
}

func (c *mockAccountConn) read() string {
  _, msg, _ := c.ReadMessage()
  return string(msg)
}

func mockServerAccount(urlChan chan string, msgChan chan string, rootWg *sync.WaitGroup, signalChan chan int8) {
  defer rootWg.Done()
  var wg sync.WaitGroup
  wg.Add(3)  // wg = 3 since wg.Done() should be called 3 times due to reconnect
  iter := 0
  upgrader := websocket.Upgrader{}
  server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    ws, _ := upgrader.Upgrade(w, r, nil)
    conn := mockAccountConn{ws}
    defer conn.Close()
    if iter == 1 {
      msgChan <- conn.read()
      conn.write(`{"stream":"authorization,","data":{"status":"unathorized","action":"authenticate"}}`)
    } else if iter == 2 {
      msgChan <- conn.read()
      conn.write(`{"stream":"authorization","data":{"status":"unauthorized","action":"listen"}}`)
      conn.write(`{"stream":"authorization","data":{"status":"authorized","action":"authenticate"}}`)
      msgChan <- conn.read()
      conn.write(`{"stream":"listening","data":{"streams":["trade_updates"]}}`)
      signalChan <- 1
    } else {
      msgChan <- conn.read()
      conn.write(`{"stream":"authorization","data":{"status":"authorized","action":"authenticate"}}`)
      msgChan <- conn.read()
      conn.write(`{"stream":"listening","data":{"streams":["trade_updates"]}}`)
      signalChan <- 1
    }
    iter++
    wg.Done()
    select {}  // Block forever to keep connection open
  }))
  defer server.Close()
  urlChan <- "ws" + strings.TrimPrefix(server.URL, "http")
  wg.Wait()
}

func TestAccountReconnect(t *testing.T) {
  assets := make(map[string]map[string]*Asset)
  assets["stock"] = make(map[string]*Asset)
  db_chan := make(chan *Query, 1)

  t.Run("pingPong error", func(t *testing.T) {
    urlChan := make(chan string)
    defer close(urlChan)
    signalChan := make(chan int8)
    defer close(signalChan)
    msgChan := make(chan string, 10)

    rootCtx, rootCancel := context.WithCancel(context.Background())

    var rootWg sync.WaitGroup
    rootWg.Add(1)

    go mockServerAccount(urlChan, msgChan, &rootWg, signalChan)

    a := NewAccount(assets, <-urlChan, db_chan)
    a.pingPong = func(wg *sync.WaitGroup, ctx context.Context, err_chan chan int8) {
      defer wg.Done()
      <-signalChan
      err_chan <-1
    }

    var wg sync.WaitGroup
    wg.Add(1)
    assert.Panics(t, func() { a.start(&wg, rootCtx, 0) }, "Expected panic after server close")
    rootCancel()

    rootWg.Wait()
    close(msgChan)

    subMsgCount := 0
    for range msgChan {
      subMsgCount++
    }

    assert.Equal(t, 5, subMsgCount)
  })

  t.Run("listen error", func(t *testing.T) {
    urlChan := make(chan string)
    defer close(urlChan)
    signalChan := make(chan int8)
    defer close(signalChan)
    msgChan := make(chan string, 10)

    rootCtx, rootCancel := context.WithCancel(context.Background())

    var rootWg sync.WaitGroup
    rootWg.Add(1)

    go mockServerAccount(urlChan, msgChan, &rootWg, signalChan)

    a := NewAccount(assets, <-urlChan, db_chan)
    a.pingPong = func(wg *sync.WaitGroup, ctx context.Context, err_chan chan int8) {
      defer wg.Done()
      <-signalChan
      err_chan <-1
    }

    var wg sync.WaitGroup
    wg.Add(1)
    assert.Panics(t, func() { a.start(&wg, rootCtx, 0) }, "Expected panic after server close")
    rootCancel()

    rootWg.Wait()
    close(msgChan)

    subMsgCount := 0
    for range msgChan {
      subMsgCount++
    }

    assert.Equal(t, 5, subMsgCount)
  })
}


type roundTripFunc func(req *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
  return f(req)
}

func init() {
  request.HttpClient = nil
  if request.HttpClient != nil {
    panic("HttpClient is not nil")
  }

  request.HttpClient = &http.Client{
    Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
      return &http.Response{ StatusCode: 200, Body: io.NopCloser(strings.NewReader(`[{"id":1}]`)) }, nil
    }),
  }

  util.Error = func(err error, details ...any) {}
  util.Warning = func(err error, details ...any) {}
  util.Ok = func(message string) {}
  util.Open = func(message string) {}
  util.Close = func(message string) {}

  push.DisablePush()
}

func TestFilterAndParse(t *testing.T) {
  a := &Account{}
  orders := map[string][]*fastjson.Value{}

  t.Run("filterRelevantOrders", func(t *testing.T) {
    globRwm.Lock()
    defer globRwm.Unlock()
    pending := map[string][]*Position{
      "BTC/USD": {
        {PositionID: "symbol[BTC/USD]_strat[rand2]_time[2025-02-24 17:22:00]"},  // Should be filtered out
        {PositionID: "symbol[BTC/USD]_strat[rand2]_time[2025-02-24 16:53:00]_close"},  // Should be filtered out
        {PositionID: "symbol[BTC/USD]_strat[rand1]_time[2025-02-24 17:20:00]_close"},
      },
      "ETH/USD": {
        {PositionID: "symbol[ETH/USD]_strat[rand2]_time[2025-02-24 17:22:00]"},  // Should be filtered out
        {PositionID: "symbol[ETH/USD]_strat[rand2]_time[2025-02-24 16:53:00]_close"},
      },
    }

    response := &http.Response{ Body: io.NopCloser(strings.NewReader(getClosedOrdersResponse)) }
    body, _ := io.ReadAll(response.Body)
    parsed, _ := fastjson.ParseBytes(body)
    arr := parsed.GetArray()

    orders = a.filterRelevantOrders(arr, pending)

    assert.Equal(t, 2, len(orders))
    assert.Equal(t, 1, len(orders["BTC/USD"]))
    assert.Equal(t, 1, len(orders["ETH/USD"]))
  })

  t.Run("parseClosedOrders", func(t *testing.T) {
    globRwm.Lock()
    parsed := a.parseClosedOrders(orders)

    assert.Equal(t, 2, len(parsed))
    assert.Equal(t, 1, len(parsed["BTC/USD"]))
    assert.Equal(t, 1, len(parsed["ETH/USD"]))

    assert.Equal(t, "ETH/USD", *parsed["ETH/USD"][0].Symbol)
    assert.Equal(t, "rand2", *parsed["ETH/USD"][0].StratName)
    assert.Equal(t, "sell", *parsed["ETH/USD"][0].Side)
    assert.Equal(t, 94897.886, *parsed["ETH/USD"][0].FilledAvgPrice)
    qty, _ := decimal.NewFromString("0.005279759")
    assert.Equal(t, qty, *parsed["ETH/USD"][0].FilledQty)
    fill_time, _ := time.Parse(time.RFC3339, string("2025-02-24T17:20:01.054777Z"))
    assert.Equal(t, fill_time, *parsed["ETH/USD"][0].FillTime)
  })
}

func TestSendCloseGTC(t *testing.T) {
  var iter int
  request.HttpClient = &http.Client{
    Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
      var ret *http.Response
      var err error
      if iter == 0 {
        ret = &http.Response{ StatusCode: 403, Body: io.NopCloser(strings.NewReader(`{"message":"forbidden."}`)) }
      } else if iter == 1 || iter == 3 {
        ret = &http.Response{ StatusCode: 200, Body: io.NopCloser(strings.NewReader(`[{"id":1}]`)) }
      } else if iter == 2 {
        ret = &http.Response{ StatusCode: 429, Body: nil }
      } else if iter == constant.REQUEST_RETRIES + 5 {
        ret = &http.Response{ StatusCode: 200, Body: nil }
      } else {
        err = errors.New("error")
      }
      iter++
      return ret, err
    }),
  }

  t.Run("Success on retry wash block status", func(t *testing.T) {
    a := &Account{}
    assert.NotPanics(t, func() { a.sendCloseGTC(decimal.NewFromFloat(0.1), "BTC/USD", 0) })
  })

  t.Run("Success on retry default case status", func(t *testing.T) {
    a := &Account{}
    assert.NotPanics(t, func() { a.sendCloseGTC(decimal.NewFromFloat(0.1), "BTC/USD", 0) })
  })

  t.Run("Fail on retry with panic", func(t *testing.T) {
    a := &Account{}
    assert.Panics(t, func() { a.sendCloseGTC(decimal.NewFromFloat(0.1), "BTC/USD", 0) })
  })
  
}

func TestUpdatePositions(t *testing.T) {
  request.HttpClient = &http.Client{
    Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
      response := &http.Response{ StatusCode: 200, Body: io.NopCloser(strings.NewReader(getAssetQtysResponse)) }
      return response, nil 
    }),
  }

  t.Run("multiple", func(t *testing.T) {
    a := &Account{
      db_chan: make(chan *Query),
      assets: make(map[string]map[string]*Asset),
    }
    a.assets["crypto"] = make(map[string]*Asset)
    a.assets["crypto"]["BTC/USD"] = &Asset{Symbol: "BTC/USD"}
    btc := a.assets["crypto"]["BTC/USD"]
    btc.Positions = make(map[string]*Position)

    strat1 := "rand1"
    strat2 := "rand2"
    strat3 := "rand3"
    btc.Positions[strat1] = &Position{ CloseOrderPending: true, OpenOrderPending: false, BadForAnalysis: false }
    btc.Positions[strat2] = &Position{ CloseOrderPending: false, OpenOrderPending: true, BadForAnalysis: false }
    btc.Positions[strat3] = &Position{ CloseOrderPending: false, OpenOrderPending: false, BadForAnalysis: false }
    parsed := map[string][]*ParsedClosedOrder{ "BTC/USD": {
      { StratName: &strat1, Symbol: &btc.Symbol },
      { StratName: &strat2, Symbol: &btc.Symbol },
    }}

    var queries []*Query
    go func() {
      defer close(a.db_chan)
      for range 2 {
        queries = append(queries, <-a.db_chan)
      }
      assert.Equal(t, 2, len(queries))
    }()

    a.updatePositions(parsed)
    assert.Equal(t, 1, len(btc.Positions))
  })
   
  t.Run("diffPositive", func(t *testing.T) {
    a := &Account{
      assets: make(map[string]map[string]*Asset),
      db_chan: make(chan *Query),
    }
    a.assets["crypto"] = make(map[string]*Asset)
    a.assets["crypto"]["BTC/USD"] = &Asset{Symbol: "BTC/USD", Positions: make(map[string]*Position)}
    btc := a.assets["crypto"]["BTC/USD"]
    btc.Qty, _ = decimal.NewFromString("0.010")

    strat1 := "rand1"
    btc.Positions[strat1] = &Position{ CloseOrderPending: false, OpenOrderPending: true, BadForAnalysis: false }
    fill_price := 95061.6
    parsed := map[string][]*ParsedClosedOrder{ "BTC/USD": {
      { StratName: &strat1, Symbol: &btc.Symbol, FilledAvgPrice: &fill_price, FillTime: &time.Time{} },
    }}

    var queries []*Query
    go func() {
      defer close(a.db_chan)
      for range 1 {
        queries = append(queries, <-a.db_chan)
      }
      assert.Equal(t, 1, len(queries))
    }()

    a.updatePositions(parsed)
    pos_qty, _ := decimal.NewFromString("0.000573338")
    assert.Equal(t, pos_qty, btc.Positions[strat1].Qty)
    btc_qty, _ := decimal.NewFromString("0.010573338")
    assert.Equal(t, btc_qty, btc.Qty)
  })

  t.Run("diffNegative", func(t *testing.T) {
    t.Run("diff equal to position qty", func(t *testing.T) {
      a := &Account{
        assets: make(map[string]map[string]*Asset),
        db_chan: make(chan *Query),
      }
      a.assets["crypto"] = make(map[string]*Asset)
      a.assets["crypto"]["BTC/USD"] = &Asset{Symbol: "BTC/USD", Positions: make(map[string]*Position)}
      btc := a.assets["crypto"]["BTC/USD"]
      btc.Qty, _ = decimal.NewFromString("0.020573338")

      strat1 := "rand1"
      pos_qty, _ := decimal.NewFromString("0.01")
      btc.Positions[strat1] = &Position{
        CloseOrderPending: true, OpenOrderPending: false, Qty: pos_qty,
        BadForAnalysis: false, NCloseOrders: 0,
      }
      fill_price := 95061.6
      parsed := map[string][]*ParsedClosedOrder{ "BTC/USD": {
        { StratName: &strat1, Symbol: &btc.Symbol, FilledAvgPrice: &fill_price, FillTime: &time.Time{} },
      }}

      var queries []*Query
      go func() {
        defer close(a.db_chan)
        for range 1 {
          queries = append(queries, <-a.db_chan)
        }
        assert.Equal(t, 1, len(queries))
      }()

      a.updatePositions(parsed)

      btc_qty, _ := decimal.NewFromString("0.010573338")
      assert.Equal(t, btc_qty, btc.Qty)
      assert.Nil(t, btc.Positions[strat1])
    })

    t.Run("diff not equal to position qty", func(t *testing.T) {
      a := &Account{
        assets: make(map[string]map[string]*Asset),
        db_chan: make(chan *Query),
      }
      a.assets["crypto"] = make(map[string]*Asset)
      a.assets["crypto"]["BTC/USD"] = &Asset{
        Symbol: "BTC/USD", 
        Positions: make(map[string]*Position),
        close: func(string, string) {},
      }
      btc := a.assets["crypto"]["BTC/USD"]
      btc.Qty, _ = decimal.NewFromString("0.020573338")

      strat1 := "rand1"
      pos_qty, _ := decimal.NewFromString("0.015")
      btc.Positions[strat1] = &Position{
        CloseOrderPending: true, OpenOrderPending: false, Qty: pos_qty,
        BadForAnalysis: false, NCloseOrders: 0,
      }
      fill_price := 95061.6
      parsed := map[string][]*ParsedClosedOrder{ "BTC/USD": {{
        StratName: &strat1, Symbol: &btc.Symbol, 
        FilledAvgPrice: &fill_price, FillTime: &time.Time{},
      }}}

      var queries []*Query
      go func() {
        defer close(a.db_chan)
        for range 1 {
          queries = append(queries, <-a.db_chan)
        }
        assert.Equal(t, 1, len(queries))
      }()

      a.updatePositions(parsed)

      btc_qty, _ := decimal.NewFromString("0.010573338")
      assert.Equal(t, btc_qty, btc.Qty)
      pos_qty, _ = decimal.NewFromString("0.005")
      assert.True(t, pos_qty.Equal(btc.Positions[strat1].Qty))
    })
  })

  t.Run("diffZero", func(t *testing.T) {
    t.Run("Open pending", func(t *testing.T) {
      a := &Account{
        assets: make(map[string]map[string]*Asset),
        db_chan: make(chan *Query),
      }
      a.assets["crypto"] = make(map[string]*Asset)
      a.assets["crypto"]["BTC/USD"] = &Asset{
        Symbol: "BTC/USD",
        Positions: make(map[string]*Position),
        close: func(string, string) {},
      }
      btc := a.assets["crypto"]["BTC/USD"]
      btc.Qty, _ = decimal.NewFromString("0.010573338")

      strat1 := "rand1"
      pos_qty, _ := decimal.NewFromString("0.015")
      btc.Positions[strat1] = &Position{
        CloseOrderPending: false, OpenOrderPending: true, Qty: pos_qty,
        BadForAnalysis: false, NCloseOrders: 0,
      }
      fill_price := 95061.6
      side := "buy"
      parsed := map[string][]*ParsedClosedOrder{ "BTC/USD": {{
        StratName: &strat1, Symbol: &btc.Symbol, Side: &side,
        FilledAvgPrice: &fill_price, FillTime: &time.Time{},
      }}}

      a.updatePositions(parsed)

      btc_qty, _ := decimal.NewFromString("0.010573338")
      assert.Equal(t, btc_qty, btc.Qty)
      assert.Nil(t, btc.Positions[strat1])
    })

    t.Run("Close pending", func(t *testing.T) {
      a := &Account{
        assets: make(map[string]map[string]*Asset),
        db_chan: make(chan *Query),
      }
      a.assets["crypto"] = make(map[string]*Asset)
      a.assets["crypto"]["BTC/USD"] = &Asset{
        Symbol: "BTC/USD",
        Positions: make(map[string]*Position),
        close: func(string, string) {},
      }
      btc := a.assets["crypto"]["BTC/USD"]
      btc.Qty, _ = decimal.NewFromString("0.010573338")

      strat1 := "rand1"
      pos_qty, _ := decimal.NewFromString("0.015")
      btc.Positions[strat1] = &Position{
        CloseOrderPending: true, OpenOrderPending: false, Qty: pos_qty,
        BadForAnalysis: false, NCloseOrders: 0,
      }
      fill_price := 95061.6
      side := "sell"
      parsed := map[string][]*ParsedClosedOrder{ "BTC/USD": {{
        StratName: &strat1, Symbol: &btc.Symbol, Side: &side,
        FilledAvgPrice: &fill_price, FillTime: &time.Time{},
      }}}

      var queries []*Query
      go func() {
        defer close(a.db_chan)
        for range 1 {
          queries = append(queries, <-a.db_chan)
        }
        assert.Equal(t, 1, len(queries))
      }()

      a.updatePositions(parsed)

      btc_qty, _ := decimal.NewFromString("0.010573338")
      assert.Equal(t, btc_qty, btc.Qty)
      assert.NotNil(t, (btc.Positions[strat1]))
    })
  })
}

var getClosedOrdersResponse = `[
  {
    "id": "a7c58468-1069-4e5f-a9c6-cfbd5f93c113",
    "client_order_id": "symbol[BTC/USD]_strat[rand2]_time[2025-02-24 17:22:00]",
    "created_at": "2025-02-24T17:22:01.045098518Z",
    "updated_at": "2025-02-24T17:22:01.106130933Z",
    "submitted_at": "2025-02-24T17:22:01.045098518Z",
    "filled_at": "null",
    "expired_at": null,
    "canceled_at": null,
    "failed_at": null,
    "replaced_at": null,
    "replaced_by": null,
    "replaces": null,
    "asset_id": "276e2673-764b-4ab6-a611-caf665ca6340",
    "symbol": "BTC/USD",
    "asset_class": "crypto",
    "notional": null,
    "qty": "0.005261602",
    "filled_qty": "0.005261602",
    "filled_avg_price": "95061.6",
    "order_class": "simple",
    "order_type": "market",
    "type": "market",
    "side": "buy",
    "position_intent": "buy_to_open",
    "time_in_force": "ioc",
    "limit_price": null,
    "stop_price": null,
    "status": "filled",
    "extended_hours": false,
    "legs": null,
    "trail_percent": null,
    "trail_price": null,
    "hwm": null,
    "subtag": null,
    "source": null
  },
  {
    "id": "2f6369f8-7570-4e00-b5ed-eafa1ca7ac93",
    "client_order_id": "symbol[BTC/USD]_strat[rand1]_time[2025-02-24 17:20:00]_close",
    "created_at": "2025-02-24T17:22:00.016839Z",
    "updated_at": "2025-02-24T17:22:00.051849Z",
    "submitted_at": "2025-02-24T17:22:00.016839Z",
    "filled_at": "2025-02-24T17:22:00.019934Z",
    "expired_at": null,
    "canceled_at": null,
    "failed_at": null,
    "replaced_at": null,
    "replaced_by": null,
    "replaces": null,
    "asset_id": "276e2673-764b-4ab6-a611-caf665ca6340",
    "symbol": "BTC/USD",
    "asset_class": "crypto",
    "notional": null,
    "qty": "0.005253887",
    "filled_qty": "0.005253887",
    "filled_avg_price": "95005.697",
    "order_class": "simple",
    "order_type": "market",
    "type": "market",
    "side": "sell",
    "position_intent": "sell_to_close",
    "time_in_force": "ioc",
    "limit_price": null,
    "stop_price": null,
    "status": "filled",
    "extended_hours": false,
    "legs": null,
    "trail_percent": null,
    "trail_price": null,
    "hwm": null,
    "subtag": null,
    "source": "access_key"
  },
  {
    "id": "dd32544b-9be0-45d3-9eed-2af7cfc3bdbb",
    "client_order_id": "symbol[ETH/USD]_strat[rand2]_time[2025-02-24 16:53:00]_close",
    "created_at": "2025-02-24T17:20:01.052404Z",
    "updated_at": "2025-02-24T17:20:01.101343Z",
    "submitted_at": "2025-02-24T17:20:01.052404Z",
    "filled_at": "2025-02-24T17:20:01.054777Z",
    "expired_at": null,
    "canceled_at": null,
    "failed_at": null,
    "replaced_at": null,
    "replaced_by": null,
    "replaces": null,
    "asset_id": "276e2673-764b-4ab6-a611-caf665ca6340",
    "symbol": "ETH/USD",
    "asset_class": "crypto",
    "notional": null,
    "qty": "0.005279759",
    "filled_qty": "0.005279759",
    "filled_avg_price": "94897.886",
    "order_class": "simple",
    "order_type": "market",
    "type": "market",
    "side": "sell",
    "position_intent": "sell_to_close",
    "time_in_force": "ioc",
    "limit_price": null,
    "stop_price": null,
    "status": "filled",
    "extended_hours": false,
    "legs": null,
    "trail_percent": null,
    "trail_price": null,
    "hwm": null,
    "subtag": null,
    "source": "access_key"
  }
]`


var getAssetQtysResponse = `[
  {
    "asset_id": "64bbff51-59d6-4b3c-9351-13ad85e3c752",
    "symbol": "BTCUSD",
    "exchange": "CRYPTO",
    "asset_class": "crypto",
    "asset_marginable": false,
    "qty": "0.010573338",
    "avg_entry_price": "94461.402922925",
    "side": "long",
    "market_value": "999.536022",
    "cost_basis": "998.772341",
    "unrealized_pl": "0.763681",
    "unrealized_plpc": "0.0007646196922468",
    "unrealized_intraday_pl": "0.763681",
    "unrealized_intraday_plpc": "0.0007646196922468",
    "current_price": "94533.63",
    "lastday_price": "95866.872",
    "change_today": "-0.0139072233419695",
    "qty_available": "0.010573338"
  },
  {
    "asset_id": "35f33a69-f5d6-4dc9-b158-4485e5e92e4b",
    "symbol": "ETHUSD",
    "exchange": "CRYPTO",
    "asset_class": "crypto",
    "asset_marginable": false,
    "qty": "0.376366581",
    "avg_entry_price": "2653.806971194",
    "side": "long",
    "market_value": "997.418135",
    "cost_basis": "998.804256",
    "unrealized_pl": "-1.386121",
    "unrealized_plpc": "-0.0013877804301226",
    "unrealized_intraday_pl": "-1.386121",
    "unrealized_intraday_plpc": "-0.0013877804301226",
    "current_price": "2650.124069478",
    "lastday_price": "2779.58",
    "change_today": "-0.0465739178300319",
    "qty_available": "0.376366581"
  },
  {
    "asset_id": "71a012ba-9ac2-4b08-9cf0-e640f4e45f6c",
    "symbol": "LINKUSD",
    "exchange": "CRYPTO",
    "asset_class": "crypto",
    "asset_marginable": false,
    "qty": "61.473846805",
    "avg_entry_price": "16.270101813",
    "side": "long",
    "market_value": "1009.400565",
    "cost_basis": "1000.185746",
    "unrealized_pl": "9.214819",
    "unrealized_plpc": "0.0092131077020968",
    "unrealized_intraday_pl": "9.214819",
    "unrealized_intraday_plpc": "0.0092131077020968",
    "current_price": "16.42",
    "lastday_price": "17.472",
    "change_today": "-0.0602106227106227",
    "qty_available": "61.473846805"
  }
]`
