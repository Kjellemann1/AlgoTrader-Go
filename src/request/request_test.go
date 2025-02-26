package request

import (
  "fmt"
  "testing"
  "net/http"
  "io"
  "strings"
  "errors"
  "github.com/shopspring/decimal"
  "github.com/stretchr/testify/assert"
  "github.com/Kjellemann1/AlgoTrader-Go/push"
  "github.com/Kjellemann1/AlgoTrader-Go/constant"
)

type roundTripFunc func(req *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
  return f(req)
}

func init() {
  HttpClient = nil
  if HttpClient != nil {
    panic("HttpClient is not nil")
  }

  push.DisablePush()
}

func TestGetPositions(t *testing.T) {
  iter := constant.REQUEST_RETRIES
  HttpClient = &http.Client{
    Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
      iter++
      if iter == 2 * constant.REQUEST_RETRIES - 1 {
        return &http.Response{ StatusCode: 200, Body: io.NopCloser(strings.NewReader(`[{"id":1}]`)) }, nil
      }
      return nil, errors.New("error")
    }),
  }

  t.Run("Success on retry", func(t *testing.T) {
    arr, err := GetPositions(0, 0)
    assert.Nil(t, err)
    assert.NotNil(t, arr)
  })

  iter = 0

  t.Run("Error on retry", func(t *testing.T) {
    arr, err := GetPositions(0, 0)
    assert.NotNil(t, err)
    assert.Nil(t, arr)
  })
}


func TestGetClosedOrders(t *testing.T) {
  iter := constant.REQUEST_RETRIES
  HttpClient = &http.Client{
    Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
      iter++
      if iter == 2 * constant.REQUEST_RETRIES - 1 {
        return &http.Response{ StatusCode: 200, Body: io.NopCloser(strings.NewReader(`[{"id":1}]`)) }, nil
      }
      return nil, errors.New("error")
    }),
  }

  t.Run("Success on retry", func(t *testing.T) {
    arr, err := GetPositions(0, 0)
    assert.Nil(t, err)
    assert.NotNil(t, arr)
  })

  iter = 0

  t.Run("Error on retry", func(t *testing.T) {
    arr, err := GetPositions(0, 0)
    assert.NotNil(t, err)
    assert.Nil(t, arr)
  })

  t.Run("Creating url", func(t *testing.T) {
    testMap := map[string]map[string]int{
      "stock": {
        "foo": 1,
      },
      "crypto": {
        "foo/bar": 1,
      },
    }
    url := urlGetClosedOrders(testMap)
    baseUrl := fmt.Sprintf(
      "%s/orders?status=closed&limit=500&direction=desc&symbols=%s",  // Max limit is 500
      constant.ENDPOINT, "foo%2Cfoo%2Fbar",
    )
    assert.Equal(t, baseUrl, url)
  })
}

func TestGetAssetQtys(t *testing.T) {
  HttpClient = &http.Client{
    Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
      return getAssetQtysResponse, nil 
    }),
  }

  resp, _ := GetAssetQtys()
  assert.Equal(t, 3, len(resp))
  btc_qty, _ := decimal.NewFromString("0.010573338")
  assert.Equal(t, btc_qty, resp["BTC/USD"])
  eth_qty, _ := decimal.NewFromString("0.376366581")
  assert.Equal(t, eth_qty, resp["ETH/USD"])
  link_qty, _ := decimal.NewFromString("61.473846805")
  assert.Equal(t, link_qty, resp["LINK/USD"])
}

var getAssetQtysResponse = &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`[
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
]`))}
