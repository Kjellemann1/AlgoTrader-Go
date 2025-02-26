package request

import (
  "io"
  "log"
  "strings"
  "time"
  "errors"
  "fmt"
  "net/http"
  "github.com/valyala/fastjson"
  "github.com/shopspring/decimal"
  "github.com/Kjellemann1/AlgoTrader-Go/constant"
  "github.com/Kjellemann1/AlgoTrader-Go/util"
)

var p = fastjson.Parser{}

var HttpClient = &http.Client{
  Timeout: constant.HTTP_TIMEOUT_SEC,
  Transport: &http.Transport{
    MaxIdleConns: 100,
    MaxIdleConnsPerHost: 50,
    IdleConnTimeout: 90 * time.Second,
    DisableKeepAlives: false,
  },
}

func GetReq(url string) ([]byte, error) {
  req, err := http.NewRequest("GET", url, nil)
  if err != nil {
    return nil, err
  }
  req.Header = constant.AUTH_HEADERS
  resp, err := HttpClient.Do(req)
  if err != nil {
    return nil, err
  }
  defer resp.Body.Close()
  body, err := io.ReadAll(resp.Body)
  if err != nil {
    fmt.Println("Error reading body. ", err)
    return nil, err
  }
  if string(body) == "[]" {
    return nil, nil
  }
  if string(body) == `{"message":"forbidden."}` {
    return nil, errors.New(string(body))
  }
  return body, nil
}

func parseBody(body []byte) ([]*fastjson.Value, error) {
  if string(body) == "[]" {
    return nil, nil
  }
  if string(body) == `{"message":"forbidden."}` {
    return nil, errors.New(string(body))
  }
  parsed, err := p.ParseBytes(body)
  if err != nil {
    return nil, err
  }
  arr := parsed.GetArray()
  if arr == nil {
    return nil, errors.New("Parsed response is not an array")
  }
  return arr, nil
}

func SendOrder(payload string) (int, error) {
  url := constant.ENDPOINT + "/orders"
  request, err := http.NewRequest("POST", url, strings.NewReader(payload))
  if err != nil {
    util.Error(err, "Request", request)
    return 0, err
  }
  request.Header = constant.AUTH_HEADERS
  response, err := HttpClient.Do(request)
  if err != nil {
    return 0, err
  }
  defer response.Body.Close()
  return response.StatusCode, nil
}

func CalculateOpenQty(asset_class string, last_price float64) decimal.Decimal {
  qty, _ := decimal.NewFromString("0")
  if asset_class == "stock" {
    qty = decimal.NewFromFloat(constant.ORDER_AMOUNT_USD / last_price).RoundDown(0)
    if qty.Cmp(decimal.NewFromInt(1)) == -1 {
      qty = decimal.NewFromInt(0)
    }
  } else if asset_class == "crypto" {
    qty = decimal.NewFromFloat(constant.ORDER_AMOUNT_USD / last_price).RoundDown(9)
  }
  return qty
}

// TODO: Make test
func GetPositions(backoff_sec int, retries int) (arr []*fastjson.Value, err error) {
  if retries >= constant.REQUEST_RETRIES {
    print("here")
    return nil, errors.New("Max retries reached. Failed to get positions.")
  }
  body, err := GetReq(constant.ENDPOINT + "/positions")
  if err != nil {
    util.Error(err, "Trying again in (seconds)", &backoff_sec)
    util.Backoff(&backoff_sec)
    retries++
    arr, err = GetPositions(backoff_sec, retries)
  } else {
    arr, err = parseBody(body)
  }
  if err != nil {
    return nil, err
  }
  return arr, nil
}

func OpenLongIOC(symbol string, asset_class string, position_id string, last_price float64) (int, error) {
  qty := CalculateOpenQty(asset_class, last_price)
  if qty.IsZero() {
    return 0, errors.New("Calculated open qty is zero")
  }

  payload := `{` +
    `"symbol": "` + symbol + `", ` +
    `"client_order_id": "` + position_id + `", ` +
    `"qty": "` + qty.String() + `", ` +
    `"side": "buy", "type": "market", "time_in_force": "ioc", "order_class": "simple"` +
  `}`

  status, err := SendOrder(payload)
  if err != nil || status != 200 {
    return status, err
  }

  return status, nil
}

// TODO: Check if position exists if order fails, and implement retry with backoff.
func CloseIOC(side string, symbol string, order_id string, qty decimal.Decimal) (int, error) {
  payload := `{` +
    `"symbol": "` + symbol + `", ` +
    `"client_order_id": "` + order_id + `_close", ` +
    `"qty": "` + qty.String() + `", ` +
    `"side": "` + side + `", ` +
    `"type": "market", "time_in_force": "ioc", "order_class": "simple"` +
  `}`

  status, err := SendOrder(payload)
  if err != nil || status != 200 {
    return status, err
  }

  return status, nil
}

func CloseGTC(side string, symbol string, order_id string, qty decimal.Decimal) (int, error) {
  payload := `{` +
    `"symbol": "` + symbol + `", ` +
    `"client_order_id": "` + order_id + `_close", ` +
    `"qty": "` + qty.String() + `", ` +
    `"side": "` + side + `", ` +
    `"type": "market", "time_in_force": "gtc", "order_class": "simple"` +
  `}`

  status, err := SendOrder(payload)
  if err != nil || status != 200 {
    if err == nil {
      err = errors.New("Bad status code")
    }
    return status, err
  }

  return status, nil
}

func CloseAllPositions(backoff_sec int, retries int) {
  if retries >= constant.REQUEST_RETRIES {
    // TODO: Check if this could be a push message
    log.Printf("[ FAIL ]\tFailed to close all positions after %d retries\n", retries)
    return
  }

  // cancel_orders=true will cancel all open orders before liquidating
  url := "https://paper-api.alpaca.markets/v2/positions?cancel_orders=true"
  req, err := http.NewRequest("DELETE", url, nil)
  if err != nil {
    util.Error(err)
    util.BackoffWithMax(&backoff_sec, 4)
    retries++
    CloseAllPositions(backoff_sec, retries)
  }
  req.Header = constant.AUTH_HEADERS

  response, err := HttpClient.Do(req)
  if err != nil {
    util.Error(err, "Response", response)
    util.BackoffWithMax(&backoff_sec, 4)
    retries++
    CloseAllPositions(backoff_sec, retries)
  }

  log.Printf("[ OK ]\tSent order to close all positions\n")
}

func urlGetClosedOrders(symbols map[string]map[string]int) (url string) {
  var symbols_str string

  ls := make([]string, 0)
  if len(symbols["stock"]) > 0 {
    for k := range symbols["stock"] {
      ls = append(ls, k)
    }
    symbols_str += strings.Join(ls, "%2C")
  }

  ls = make([]string, 0)
  if len(symbols_str) > 0 && len(symbols["crypto"]) > 0 {
    symbols_str += "%2C"
    for k := range symbols["crypto"] {
        ls = append(ls, k)
    }
    symbols_str += strings.Replace(strings.Join(ls, "%2C"), "/", "%2F", len(symbols["crypto"]))
  }

  // A better solution than just taking the last 500 orders could be to only take the
  // ones since the time of the last executed trade from the database.
  url = fmt.Sprintf(
    "%s/orders?status=closed&limit=500&direction=desc&symbols=%s",  // Max limit is 500
    constant.ENDPOINT, symbols_str,
  )
  return
}

func GetClosedOrders(symbols map[string]map[string]int, backoff_sec int, retries int) (parsed []*fastjson.Value, err error) {
  if retries >= 4 {
    return nil, errors.New("Max retries reached. Failed to get closed orders.")
  }

  url := urlGetClosedOrders(symbols)

  body, err := GetReq(url)
  if err != nil || body == nil {
    util.Error(err,
      "body", string(body),
      "Trying again in (seconds)", &backoff_sec,
    )
    util.Backoff(&backoff_sec)
    retries++
    parsed, err = GetClosedOrders(symbols, backoff_sec, retries)
  } else {
    parsed, err = parseBody(body)
  }

  if err != nil {
    return nil, err
  }

  if retries > 0 {
    log.Printf("[ OK ]\tRetrieved closed orders after (%d) retries\n", retries)
  }

  return parsed, nil
}

// TODO: Make test
func GetAssetQtys() (qtys map[string]decimal.Decimal, err error) {
  apos, err := GetPositions(5, 0)
  if err != nil {
    return
  }

  qtys = make(map[string]decimal.Decimal)
  var crypto bool
  for _, v  := range apos {
    qty, err := decimal.NewFromString(string(v.GetStringBytes("qty")))
    if err != nil {
      return nil, err
    }
    crypto = false
    for _, s := range constant.CRYPTO_SYMBOLS {
      if strings.Replace(s, "/", "", 1) == string(v.GetStringBytes("symbol")) {
        qtys[s] = qty
        crypto = true
      }
    }
    if !crypto {
      qtys[string(v.GetStringBytes("symbol"))] = qty
    }
  }

  return
}
