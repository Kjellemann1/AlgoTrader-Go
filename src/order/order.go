
package order

import (
  "io"
  "log"
  "fmt"
  "bytes"
  "strings"
  "net/http"
  "github.com/valyala/fastjson"
  "github.com/shopspring/decimal"

  "github.com/Kjellemann1/AlgoTrader-Go/src/constant"
  "github.com/Kjellemann1/AlgoTrader-Go/src/util/push"
  "github.com/Kjellemann1/AlgoTrader-Go/src/util/backoff"
)


func SendOrder(payload string) error {
  url := fmt.Sprintf("%s/orders", constant.ENDPOINT)
  request, err := http.NewRequest("POST", url, bytes.NewBuffer([]byte(payload)))
  if err != nil {
    fmt.Println("Error making POST request:", err)
  }
  request.Header = constant.AUTH_HEADERS
  response, err := http.DefaultClient.Do(request)
  fmt.Println("Order sent")
  if err != nil {
    log.Printf(
      "[ ERROR ]\tMaking POST request failed:" +
      "  -> Request: %v+\n" +
      "  -> Response: %v+\n" +
      "  -> Error: %s\n",
    request, response, err)
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
  url := fmt.Sprintf("%s/positions", constant.ENDPOINT)
  qty, _ := decimal.NewFromString("0")
  var p = fastjson.Parser{}
  for {
    // Create request
    req, err := http.NewRequest("GET", url, nil)
    if err != nil {
      log.Printf(
        "[ WARNING ]\tChecking if position exsists failed when sending request.\n" +
        "  -> Request: %+v\n  -> Error: %s\n  -> Trying again in %d seconds", 
        req, err, &backoff_sec,
      )
      push.Error("Failed to create request when checking if position exists.", err)
      backoff.BackoffWithMax(&backoff_sec, backoff_max_sec)
      continue
    }
    // Set headers
    req.Header = constant.AUTH_HEADERS
    // Send request
    resp, err := http.DefaultClient.Do(req)
    if err != nil {
      log.Printf(
        "[ WARNING ]\tChecking if position exists failed when sending request.\n" +
        "  -> Response: %+v\n  -> Error: %s\n  -> Trying again in %d seconds", 
        resp, err, &backoff_sec,
      )
      push.Error("Failed to send request when checking if position exists.", err)
      backoff.BackoffWithMax(&backoff_sec, backoff_max_sec)
      continue
    }
    defer resp.Body.Close()
    // Read response body
    body, err := io.ReadAll(resp.Body)
    if err != nil {
      log.Printf(
        "[ WARNING ] Checking if positions exists failed.\n" +
        "  -> Body: %s\n  -> Error: %s\n" + "  -> Trying again in %d seconds",
        body, err, &backoff_sec,
      )
      push.Error("Failed to read response body when checking if position exists.", err)
      backoff.BackoffWithMax(&backoff_sec, backoff_max_sec)
      continue
    }
    // Parse response body
    parsed, err := p.ParseBytes(body)
    if err != nil {
      log.Printf(
        "[ WARNING ] Parsing response in CheckIfPositionExists() failed.\n" +
        "  -> Parsed: %s\n  -> Error: %s\n  -> Trying again in %d seconds", 
        parsed, err, &backoff_sec,
      )
      push.Error("Failed to parse response body when checking if position exists.", err)
      backoff.BackoffWithMax(&backoff_sec, backoff_max_sec)
      continue
    }
    // Get array returns nil if the array is empty, so need to check that before trying to get the array
    if string(body) == "[]" {
      return false, qty
    } else if string(body) == `{"message":"forbidden."}` {
      log.Printf(
        "[ WARNING ] Forbidden response in CheckIfPositionExists().\n" +
        "  -> Make sure HTTP headers are correct.\n  -> Body: %s\n  -> Trying again in %d seconds",
        string(body), &backoff_sec,
      )
      push.Error("Forbidden response when checking if position exists.", nil)
      backoff.BackoffWithMax(&backoff_sec, backoff_max_sec)
      continue
    }
    // Get array from response
    arr := parsed.GetArray()
    if arr == nil {
      log.Printf(
        "[ WARNING ] Failed to get array from response in CheckIfPositionExists().\n" +
        "  -> Response: %s\n  -> Trying again in %d seconds", string(body), &backoff_sec,
      )
      push.Error("Failed to get array from response body when checking if position exists.", nil)
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


func CheckIfQtyMatches(symbol string, qty decimal.Decimal) {
  // TODO
}
