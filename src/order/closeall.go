
package order

import (
  "net/http"
  "log"

  "github.com/Kjellemann1/AlgoTrader-Go/src/util/backoff"
  "github.com/Kjellemann1/AlgoTrader-Go/src/util/push"
  "github.com/Kjellemann1/AlgoTrader-Go/src/util/pretty"
  "github.com/Kjellemann1/AlgoTrader-Go/src/constant"
)


// TDOD: Also implement a function like this for clearing positions table in database
func CloseAllPositions(backoff_sec int, retries int) {
  url := "https://paper-api.alpaca.markets/v2/positions?cancel_orders=true"
  req, err := http.NewRequest("DELETE", url, nil)
  if err != nil {
    log.Printf(
      "[ ERROR ]\tFailed to create request in CloseAllPositions()\n" +
      "  -> Request: %s\n  -> Error: %s\n", pretty.RequestToString(req), err,
    )
    push.Error("Error creating request in CloseAllPositions()", err)
    backoff.BackoffWithMax(&backoff_sec, 4)
    CloseAllPositions(backoff_sec, retries)
  }
  req.Header = constant.AUTH_HEADERS
  response, err := http.DefaultClient.Do(req)
  if err != nil {
    log.Printf(
      "[ ERROR ]\tFailed to send request in CloseAllPositions()\n" +
      "  -> Response: %s\n  -> Error: %s\n", pretty.ResponseToString(response), err,
    )
    push.Error("Error sending request in CloseAllPositions()", err)
    backoff.BackoffWithMax(&backoff_sec, 4)
    CloseAllPositions(backoff_sec, retries)
  }
  log.Printf("[ OK ]\tSent order to close all positions\n")
}
