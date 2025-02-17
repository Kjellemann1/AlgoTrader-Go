package src

import (
  "math/rand"
  // "github.com/markcheno/go-talib"
  // "github.com/Kjellemann1/Gostuff/indicators"
)

// func (a *Asset) EMACrossRSI() {
//   a.Mutex.Lock()
//   strat_name := "EMACrossRSI"
//   rsi := talib.Rsi(a.C, 13)
//   fast := talib.Ema(a.C, 6)
//   slow := talib.Ema(a.C, 12)
//   if fast[a.I(0)] > slow[a.I(0)] {
//     if fast[a.I(1)] < slow[a.I(1)] {
//       if rsi[a.I(0)] < 15 {
//         a.Close("IOC", strat_name)
//       }
//     }
//   }
//   if fast[a.I(0)] < slow[a.I(0)] {
//     if fast[a.I(1)] > slow[a.I(1)] {
//       if rsi[a.I(0)] > 70 {
//         a.Open("long", "IOC", strat_name)
//       }
//     }
//   }
//   a.Mutex.Unlock()
// }

// func (a *Asset) RSIMiddle() {
//   a.Mutex.Lock()
//   strat_name := "RSIMiddle"
//   rsi := talib.Rsi(a.C, 70)
//   _, mid, _ := indicators.Middle(a.H, a.L, .5, .5, 360)
//   if rsi[a.I(1)] > 35 {
//     if rsi[a.I(0)] < 35 {
//       if a.C[a.I(0)] < mid {
//         a.Open("long", "IOC", strat_name)
//       }
//     }
//   }
//   a.TakeProfit(1, strat_name)
//   a.StopLoss(5, strat_name)
//   a.Mutex.Unlock()
// }

func (a *Asset) testRand() {
  a.Mutex.Lock()
  strat_name := "testRand"
  num := rand.Intn(100)
  if num < 20 {
    a.Open("long", "IOC", strat_name)
  }
  if num >= 80 {
    a.Close("IOC", strat_name)
  }
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
