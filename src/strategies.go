
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


func (a *Asset) testingRand1() {
  a.mutex.Lock()
  strat_name := "testingRand1_3"
  num := rand.Intn(100)
  if num < 20 {
    a.OpenPosition("long", "IOC", strat_name)
  } else if num >= 80 {
    a.ClosePosition("IOC", strat_name)
  }
  a.mutex.Unlock()
}


func (a *Asset) testingRand2() {
  a.mutex.Lock()
  strat_name := "testingRand2_3"
  num := rand.Intn(100)
  if num < 20 {
    a.OpenPosition("long", "IOC", strat_name)
  } else if num >= 80 {
    a.ClosePosition("IOC", strat_name)
  }
  a.mutex.Unlock()
}
