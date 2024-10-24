
package src

import (
  "github.com/Kjellemann1/AlgoTrader-Go/src/indicator"
)

//
// func (a *Asset) testingStrategy() {
//   if _, ok := a.Positions["test"]; !ok {
//     a.OpenPosition("long", "IOC", "test")
//   } else {
//     a.ClosePosition("IOC", "test")
//   }
// }

// TODO: This strat does not work until filling of windows with historical data at start is implemented
func (a *Asset) testingStrategy() {
  strat_name := "test"

  rsi := indicator.RSI(a.Close, 14)

  if rsi[len(rsi)-1] > 55 {
    a.OpenPosition("short", "IOC", strat_name)
  }

  if rsi[len(rsi)-1] < 45 {
    a.ClosePosition("IOC", strat_name)
  }
}
