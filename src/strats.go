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
