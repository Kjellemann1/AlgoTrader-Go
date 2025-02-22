package main

import (
  "math/rand"
  // "github.com/markcheno/go-talib"
  // "github.com/Kjellemann1/Gostuff/indicators"
)

func (a *Asset) rand() {
  a.Mutex.Lock()

  num1 := rand.Intn(100)
  num2 := rand.Intn(100)

  if num1 < 5 {
    a.open("long", "IOC", "rand1")
    a.close("IOC", "rand2")
  } else if num1 >= 95 {
    a.close("IOC", "rand1")
    a.open("long", "IOC", "rand2")
  }

  if num2 < 5 {
    a.open("long", "IOC", "rand3")
    a.close("IOC", "rand4")
  } else if num1 >= 95 {
    a.close("IOC", "rand3")
    a.open("long", "IOC", "rand4")
  }

  a.Mutex.Unlock()
}
