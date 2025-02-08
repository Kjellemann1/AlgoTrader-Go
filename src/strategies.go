
package src

import (
  // "math/rand"

  "github.com/markcheno/go-talib"
)


func (a *Asset) testCool() {
  a.Mutex.Lock()
  strat_name := "testCool_1"
  rsi := talib.Rsi(a.C[:], 17)
  if a.IndexSingle(&rsi, 0) < 30 {
    if a.IndexSingle(&rsi, 1) < 30 {
      a.Open("long", "IOC", strat_name)
    }
  } 
  a.StopLoss(5, strat_name)
  a.TakeProfit(5, strat_name)
  a.Mutex.Unlock()
}


// func (a *Asset) testRSI() {
//   a.Mutex.Lock()
//   strat_name := "test2_RSI"
//   rsi := talib.Rsi(a.C[:], 14)
//   if a.IndexSingle(&rsi, 0) > 70 {
//     a.Open("long", "IOC", strat_name)
//   }
//   if rsi[len(rsi)-1] < 30 {
//     a.Close("IOC", strat_name)
//   }
//   a.StopLoss(5, strat_name)
//   a.TakeProfit(5, strat_name)
//   a.Mutex.Unlock()
// }
//
//
// func (a *Asset) testSMA() {
//   a.Mutex.Lock()
//   strat_name := "test2_SMA"
//   fast := talib.Sma(a.C[:], 20)
//   slow := talib.Sma(a.C[:], 75)
//   if a.IndexSingle(&fast, 0) > a.IndexSingle(&slow, 0) {
//     if a.IndexSingle(&fast, 1) < a.IndexSingle(&slow, 1) {
//       if a.IndexSingle(&fast, 2) < a.IndexSingle(&slow, 2) {
//         if a.IndexSingle(&fast, 3) < a.IndexSingle(&slow, 3) {
//           a.Open("long", "IOC", strat_name)
//         }
//       }
//     }
//   }
//   if a.IndexSingle(&fast, 0) < a.IndexSingle(&slow, 0) {
//     if a.IndexSingle(&fast, 1) > a.IndexSingle(&slow, 1) {
//       if a.IndexSingle(&fast, 2) > a.IndexSingle(&slow, 2) {
//         if a.IndexSingle(&fast, 3) > a.IndexSingle(&slow, 3) {
//           a.Close("IOC", strat_name)
//         }
//       }
//     }
//   }
//   a.StopLoss(5, strat_name)
//   a.TakeProfit(5, strat_name)
//   a.Mutex.Unlock()
// }
//
//
// func (a *Asset) testBBands() {
//   a.Mutex.Lock()
//   strat_name := "test2_BBands"
//   upper, _, lower := talib.BBands(a.C[:], 20, 2, 2, 0)
//   currentPrice := a.IndexSingle(&a.C, 0)
//   if currentPrice < a.IndexSingle(&lower, 0) {
//     a.Open("long", "IOC", strat_name)
//   }
//   if currentPrice > a.IndexSingle(&upper, 0) {
//     a.Close("IOC", strat_name)
//   }
//   a.StopLoss(5, strat_name)
//   a.TakeProfit(5, strat_name)
//   a.Mutex.Unlock()
// }
//
//
// func (a *Asset) testMomentum() {
//   a.Mutex.Lock()
//   strat_name := "test2_Momentum"
//   momentum := a.IndexSingle(&a.C, 0) - a.IndexSingle(&a.C, 5)
//   threshold := 1.0
//   if momentum > threshold {
//     a.Open("long", "IOC", strat_name)
//   }
//   if momentum < -threshold {
//     a.Close("IOC", strat_name)
//   }
//   a.StopLoss(5, strat_name)
//   a.TakeProfit(5, strat_name)
//   a.Mutex.Unlock()
// }
//
//
// func (a *Asset) testRand1() {
//   a.Mutex.Lock()
//   strat_name := "testRand1_2"
//   num := rand.Intn(100)
//   if num < 5 {
//     a.Open("long", "IOC", strat_name)
//   } else if num >= 95 {
//     a.Close("IOC", strat_name)
//   }
//   a.Mutex.Unlock()
// }
//
//
// func (a *Asset) testRand2() {
//   a.Mutex.Lock()
//   strat_name := "testRand2_2"
//   num := rand.Intn(100)
//   if num < 5 {
//     a.Open("long", "IOC", strat_name)
//   } else if num >= 95 {
//     a.Close("IOC", strat_name)
//   }
//   a.Mutex.Unlock()
// }
