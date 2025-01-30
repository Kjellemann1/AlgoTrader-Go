
package order

import (
  "io"
  "log"
  "strings"
  "time"
  "net/http"
  "github.com/valyala/fastjson"
  "github.com/shopspring/decimal"

  "github.com/Kjellemann1/AlgoTrader-Go/src/constant"
  "github.com/Kjellemann1/AlgoTrader-Go/src/util/backoff"
  "github.com/Kjellemann1/AlgoTrader-Go/src/util/handlelog"
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
    handlelog.Error(err, "Request", request)
    return err
  }
  request.Header = constant.AUTH_HEADERS
  dnsStart := time.Now()
  response, err := httpClient.Do(request)
  dnsDuration := time.Since(dnsStart)
  log.Printf(
    "[ TIMER server ]\t%.3f\n",
    dnsDuration.Seconds(),
  )
  if err != nil {
    handlelog.Error(err, response)
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
    qty = decimal.NewFromFloat(constant.ORDER_AMOUNT_USD / last_price).RoundDown(9)  // Alpaca accepts max 9 decimal places
  }
  return qty
}


// TODO: Make sure this doesn't run for an infinite loop by implementing a max nuber of retries
// Check if position exists. Used for errors in close orders
func CheckIfPositionExists(symbol string) (bool, decimal.Decimal) {
  // TODO: Specific error handling
  var backoff_sec int = 4
  var backoff_max_sec int = 60
  stripped_symbol := strings.Replace(symbol, "/", "", 1)
  url := constant.ENDPOINT + "/positions"
  qty, _ := decimal.NewFromString("0")
  var p = fastjson.Parser{}
  for {
    // Create request
    req, err := http.NewRequest("GET", url, nil)
    if err != nil {
      handlelog.Error(err, "Request", req)
      backoff.BackoffWithMax(&backoff_sec, backoff_max_sec)
      continue
    }
    // Set headers
    req.Header = constant.AUTH_HEADERS
    // Send request
    resp, err := httpClient.Do(req)
    if err != nil {
      handlelog.Warning(err, "Response", resp, "Trying again in (seconds)", &backoff_sec)
      backoff.BackoffWithMax(&backoff_sec, backoff_max_sec)
      continue
    }
    defer resp.Body.Close()
    // Read response body
    body, err := io.ReadAll(resp.Body)
    if err != nil {
      handlelog.Warning(err, "Response", resp, "Trying again in (seconds)", &backoff_sec)
      backoff.BackoffWithMax(&backoff_sec, backoff_max_sec)
      continue
    }
    // Parse response body
    parsed, err := p.ParseBytes(body)
    if err != nil {
      handlelog.Warning(err, "Parsed", parsed, "Trying again in (seconds)", &backoff_sec)
      backoff.BackoffWithMax(&backoff_sec, backoff_max_sec)
      continue
    }
    // Get array returns nil if the array is empty, so need to check that before trying to get the array
    if string(body) == "[]" {
      return false, qty
    } else if string(body) == `{"message":"forbidden."}` {
      handlelog.Warning(err, "message", string(body), "Trying again in (seconds)", &backoff_sec)
      backoff.BackoffWithMax(&backoff_sec, backoff_max_sec)
      continue
    }
    // Get array from response
    arr := parsed.GetArray()
    if arr == nil {
      handlelog.Warning(err, "Response", string(body), "Trying again in (seconds)", &backoff_sec)
      backoff.BackoffWithMax(&backoff_sec, backoff_max_sec)
      continue
    }
    // Check if position exists
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
}


func CheckIfOrderExists(symbol string) (bool, string, error) {
  url := constant.ENDPOINT + "/orders?status=open&symbols=" + symbol
  request, err := http.NewRequest("GET", url, nil)
  if err != nil {handlelog.Error(err, "Request", request, "URL", url); return false, "", err}

  request.Header = constant.AUTH_HEADERS
  response, err := httpClient.Do(request)
  if err != nil {handlelog.Error(err, "Request", request, "Response", response); return false, "", err}
  defer response.Body.Close()

  body, err := io.ReadAll(response.Body)
  if err != nil {handlelog.Error(err, "Response", response); return false, "", err}

  var p = fastjson.Parser{}
  parsed, err := p.ParseBytes(body)
  if err != nil {handlelog.Error(err, "Body", string(body)); return false, "", err}

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
    handlelog.Error(err, "Request", req)
    return err
  }
  req.Header = constant.AUTH_HEADERS
  resp, err := httpClient.Do(req)
  if err != nil {
    handlelog.Error(err, "Request", req, "Response", resp)
    return err
  }
  defer resp.Body.Close()
  return nil
}


func CheckIfQtyMatches(symbol string, qty decimal.Decimal) {
  // TODO
}
