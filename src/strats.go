package main

import (
  "math/rand"
  // "github.com/markcheno/go-talib"
  // "github.com/Kjellemann1/Gostuff/indicators"
)

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
