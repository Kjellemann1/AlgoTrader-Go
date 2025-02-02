
package src

import (
  "math/rand"

  // "github.com/Kjellemann1/AlgoTrader-Go/src/indicator"
)


// func (a *Asset) testingStrategy() {
//   a.mutex.Lock()
//   strat_name := "test9"
//   rsi := indicator.RSI(a.Close, 14)
//   if rsi[len(rsi)-1] > 70 {
//     a.OpenPosition("long", "IOC", strat_name)
//   }
//   if rsi[len(rsi)-1] < 30 {
//     a.ClosePosition("IOC", strat_name)
//   }
//   a.mutex.Unlock()
// }


// func (a *Asset) testingStrategy() {
//   a.mutex.Lock()
//   if _, ok := a.Positions["test"]; !ok {
//     a.OpenPosition("long", "IOC", "test")
//   } else {
//     a.ClosePosition("IOC", "test")
//   }
//   a.mutex.Unlock()
// }


func (a *Asset) testingRand() {
  a.mutex.Lock()
  strat_name := "testingRand9"
  num := rand.Intn(100)
  if num < 40 {
    a.OpenPosition("long", "IOC", strat_name)
  } else if num > 60 {
    a.ClosePosition("IOC", strat_name)
  }
  a.mutex.Unlock()
}
