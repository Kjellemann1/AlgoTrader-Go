package main

import (
  "io"
  "time"
  "strings"
  "testing"
  "net/http"
  "github.com/shopspring/decimal"
  "github.com/valyala/fastjson"
  "github.com/stretchr/testify/assert"
  "github.com/Kjellemann1/AlgoTrader-Go/request"
  "github.com/Kjellemann1/AlgoTrader-Go/push"
)

func init() {
  request.HttpClient = nil
  if request.HttpClient != nil {
    panic("HttpClient is not nil")
  }

  push.DisablePush()
}

func TestCheckPending(t *testing.T) {
  a := &Account{}
  orders := map[string][]*fastjson.Value{}

  t.Run("filterRelevantOrders", func(t *testing.T) {
    pending := map[string][]*Position{
      "BTC/USD": {
        {PositionID: "symbol[BTC/USD]_strat[rand2]_time[2025-02-24 17:22:00]"},
        {PositionID: "symbol[BTC/USD]_strat[rand1]_time[2025-02-24 17:20:00]_close"},
        {PositionID: "symbol[BTC/USD]_strat[rand2]_time[2025-02-24 16:53:00]_close"},  // Should be filtered out
      },
      "ETH/USD": {
        {PositionID: "symbol[ETH/USD]_strat[rand2]_time[2025-02-24 16:53:00]_close"},
        {PositionID: "symbol[ETH/USD]_strat[rand2]_time[2025-02-24 17:22:00]"},  // Should be filtered out
      },
    }

    body, _ := io.ReadAll(response.Body)
    parsed, _ := fastjson.ParseBytes(body)
    arr := parsed.GetArray()

    orders = a.filterRelevantOrders(arr, pending)

    assert.Equal(t, 2, len(orders))
    assert.Equal(t, 2, len(orders["BTC/USD"]))
    assert.Equal(t, 1, len(orders["ETH/USD"]))
  })

  t.Run("parseClosedOrders", func(t *testing.T) {
    parsed := a.parseClosedOrders(orders)

    assert.Equal(t, 2, len(parsed))
    assert.Equal(t, 2, len(parsed["BTC/USD"]))
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

  t.Run("updatePositions", func(t *testing.T) {

  })
}

var response = http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`[
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
]`))}
