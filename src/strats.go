package main

import (
  "math/rand"
  // "github.com/markcheno/go-talib"
  // "github.com/Kjellemann1/Gostuff/indicators"
)

func (a *Asset) rand() {
  a.Mutex.Lock()

  prob := 20

  num1 := rand.Intn(100)
  num2 := rand.Intn(100)

  if num1 < prob {
    a.open("long", "IOC", "rand1")
    a.close("IOC", "rand2")
  } else if num1 >= (100 - prob) {
    a.close("IOC", "rand1")
    a.open("long", "IOC", "rand2")
  }

  if num2 < prob {
    a.open("long", "IOC", "rand3")
    a.close("IOC", "rand4")
  } else if num1 >= (100 - prob) {
    a.close("IOC", "rand3")
    a.open("long", "IOC", "rand4")
  }

  a.Mutex.Unlock()
}
