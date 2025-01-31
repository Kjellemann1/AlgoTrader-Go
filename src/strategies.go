
package src

import (
  "github.com/Kjellemann1/AlgoTrader-Go/src/indicator"
)


func (a *Asset) testingStrategy() {
  strat_name := "test5"
  rsi := indicator.RSI(a.Close, 14)
  if rsi[len(rsi)-1] > 70 {
    a.OpenPosition("long", "IOC", strat_name)
  }
  if rsi[len(rsi)-1] < 30 {
    a.ClosePosition("IOC", strat_name)
  }
}


// func (a *Asset) testingStrategy() {
//   if _, ok := a.Positions["test"]; !ok {
//     a.OpenPosition("long", "IOC", "test")
//   } else {
//     a.ClosePosition("IOC", "test")
//   }
// }
