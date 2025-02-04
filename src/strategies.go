
package src

import (
  // "math/rand"

  "github.com/Kjellemann1/AlgoTrader-Go/src/indicator"
)


func (a *Asset) testingRSI1() {
  a.mutex.Lock()
  strat_name := "testingRSI1_1"
  rsi := indicator.RSI(a.Close, 14)
  if rsi[len(rsi)-1] > 80 {
    a.OpenPosition("long", "IOC", strat_name)
  }
  if rsi[len(rsi)-1] < 20 {
    a.ClosePosition("IOC", strat_name)
  }
  a.mutex.Unlock()
}


func (a *Asset) testingRSI2() {
  a.mutex.Lock()
  strat_name := "testingRSI2_1"
  rsi := indicator.RSI(a.Close, 14)
  if rsi[len(rsi)-1] > 85 {
    a.OpenPosition("long", "IOC", strat_name)
  }
  if rsi[len(rsi)-1] < 15 {
    a.ClosePosition("IOC", strat_name)
  }
  a.mutex.Unlock()
}


func (a *Asset) testingRSI3() {
  a.mutex.Lock()
  strat_name := "testingRSI3_1"
  rsi := indicator.RSI(a.Close, 14)
  if rsi[len(rsi)-1] > 90 {
    a.OpenPosition("long", "IOC", strat_name)
  }
  if rsi[len(rsi)-1] < 10 {
    a.ClosePosition("IOC", strat_name)
  }
  a.mutex.Unlock()
}


// func (a *Asset) testingRand1() {
//   a.mutex.Lock()
//   strat_name := "testingRand1_1"
//   num := rand.Intn(100)
//   if num < 5 {
//     a.OpenPosition("long", "IOC", strat_name)
//   } else if num >= 95 {
//     a.ClosePosition("IOC", strat_name)
//   }
//   a.mutex.Unlock()
// }


// func (a *Asset) testingRand2() {
//   a.mutex.Lock()
//   strat_name := "testingRand2_1"
//   num := rand.Intn(100)
//   if num < 5 {
//     a.OpenPosition("long", "IOC", strat_name)
//   } else if num >= 95 {
//     a.ClosePosition("IOC", strat_name)
//   }
//   a.mutex.Unlock()
// }
