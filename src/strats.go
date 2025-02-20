package main

import (
  "math/rand"
  // "github.com/markcheno/go-talib"
  // "github.com/Kjellemann1/Gostuff/indicators"
)

func (a *Asset) rand() {
  a.Mutex.Lock()
  strat_name := "rand"
  num := rand.Intn(100)

  if num < 10 {
    a.Open("long", "IOC", strat_name)
  }

  if num >= 90 {
    a.Close("IOC", strat_name)
  }

  a.Mutex.Unlock()
}
