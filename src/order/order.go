package order

import (
  "io"
  "strings"
  "time"
  "errors"
  "net/http"
  "github.com/valyala/fastjson"
  "github.com/shopspring/decimal"

  "github.com/Kjellemann1/AlgoTrader-Go/constant"
  "github.com/Kjellemann1/AlgoTrader-Go/util"
)

var httpClient = &http.Client{
  Timeout: constant.HTTP_TIMEOUT_SEC,
  Transport: &http.Transport{
    MaxIdleConns: 100,
    MaxIdleConnsPerHost: 50,
    IdleConnTimeout: 90 * time.Second,
    DisableKeepAlives: false,
  },
}

func SendOrder(payload string) error {
  url := constant.ENDPOINT + "/orders"
  request, err := http.NewRequest("POST", url, strings.NewReader(payload))
  if err != nil {
    util.Error(err, "Request", request)
    return err
  }
  request.Header = constant.AUTH_HEADERS
  response, err := httpClient.Do(request)
  if err != nil {
    util.Error(err, response)
    return err
  }
  defer response.Body.Close()
  return nil
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


func sendPosRequest() (*http.Response, error) {
  url := constant.ENDPOINT + "/positions"
  req, err := http.NewRequest("GET", url, nil)
  if err != nil {
    return nil, err
  }
  req.Header = constant.AUTH_HEADERS
  resp, err := httpClient.Do(req)
  if err != nil {
    return nil, err
  }
  return resp, nil
}

func parsePosResponse(resp *http.Response) ([]*fastjson.Value, error) {
  var p = fastjson.Parser{}
  body, err := io.ReadAll(resp.Body)
  if err != nil {
    return nil, err
  }
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

func checkIfPosExists(arr []*fastjson.Value, symbol string) (bool, decimal.Decimal) {
  stripped_symbol := strings.Replace(symbol, "/", "", 1)
  qty, _ := decimal.NewFromString("0")
    for _, v := range arr {
      if string(v.GetStringBytes("symbol")) == stripped_symbol {
        qty, err := decimal.NewFromString(string(v.GetStringBytes("qty")))
        if err != nil {
          qty, _ = decimal.NewFromString("0")
        }
        return true, qty
      }
    }
    return false, qty
}

func CheckIfPositionExists(symbol string) (bool, decimal.Decimal) {
  backoff_sec := 4
  backoff_max_sec := 60
  retries := 0
  for {
    if retries >= 3 {
      util.Error(errors.New("Max retries reached for CheckIfPositionExists"), 
        "Symbol", symbol, "Setting No_new_positions = true", "...",
      )
      return false, decimal.NewFromInt(0)
    }
    resp, err := sendPosRequest()
    if err != nil {
      util.Error(err, "Trying again in (seconds)", &backoff_sec)
      util.BackoffWithMax(&backoff_sec, backoff_max_sec)
      retries++
      continue
    }
    defer resp.Body.Close()
    arr, err := parsePosResponse(resp)
    if err != nil {
      util.Error(err, "Trying again in (seconds)", &backoff_sec)
      util.BackoffWithMax(&backoff_sec, backoff_max_sec)
      retries++
      continue
    } else if arr == nil {
      return false, decimal.NewFromInt(0)
    }
    return checkIfPosExists(arr, symbol)
  }
}

func CheckIfOrderExists(symbol string) (bool, string, error) {
  url := constant.ENDPOINT + "/orders?status=open&symbols=" + symbol
  request, err := http.NewRequest("GET", url, nil)
  if err != nil {util.Error(err, "Request", request, "URL", url); return false, "", err}

  request.Header = constant.AUTH_HEADERS
  response, err := httpClient.Do(request)
  if err != nil {util.Error(err, "Request", request, "Response", response); return false, "", err}
  defer response.Body.Close()

  body, err := io.ReadAll(response.Body)
  if err != nil {util.Error(err, "Response", response); return false, "", err}

  var p = fastjson.Parser{}
  parsed, err := p.ParseBytes(body)
  if err != nil {util.Error(err, "Body", string(body)); return false, "", err}

  arr := parsed.GetArray()
  if len(arr) == 0 {
    return false, "", nil
  }
  var order_id string
  for _, v := range arr {
    if string(v.GetStringBytes("symbol")) == symbol {
      order_id = string(v.GetStringBytes("id"))
    }
  }
  return true, order_id, nil
}

func CancelOrder(order_id string) error {
  url := constant.ENDPOINT + "/orders/" + order_id
  req, err := http.NewRequest("DELETE", url, nil)
  if err != nil {
    util.Error(err, "Request", req)
    return err
  }
  req.Header = constant.AUTH_HEADERS
  resp, err := httpClient.Do(req)
  if err != nil {
    util.Error(err, "Request", req, "Response", resp)
    return err
  }
  defer resp.Body.Close()
  return nil
}

func CheckIfQtyMatches(symbol string, qty decimal.Decimal) {
  // TODO
}
