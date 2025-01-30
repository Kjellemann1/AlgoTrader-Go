
package order

import (
  "net/http"
  "log"

  "github.com/Kjellemann1/AlgoTrader-Go/src/util/backoff"
  "github.com/Kjellemann1/AlgoTrader-Go/src/constant"
  "github.com/Kjellemann1/AlgoTrader-Go/src/util/handlelog"
)


// TDOD: Also implement a function like this for clearing positions table in database
// TODO: Not sure this is implemented correctly?
func CloseAllPositions(backoff_sec int, retries int) {
  url := "https://paper-api.alpaca.markets/v2/positions?cancel_orders=true"
  req, err := http.NewRequest("DELETE", url, nil)
  if err != nil {
    handlelog.Error(err)
    backoff.BackoffWithMax(&backoff_sec, 4)
    CloseAllPositions(backoff_sec, retries)
  }
  req.Header = constant.AUTH_HEADERS
  response, err := http.DefaultClient.Do(req)
  if err != nil {
    handlelog.Error(err, "Response", response)
    backoff.BackoffWithMax(&backoff_sec, 4)
    CloseAllPositions(backoff_sec, retries)
  }
  log.Printf("[ OK ]\tSent order to close all positions\n")
}
