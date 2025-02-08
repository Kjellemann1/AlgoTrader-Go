
package test

import (
  "github.com/Kjellemann1/AlgoTrader-Go/src"
  "github.com/Kjellemann1/AlgoTrader-Go/src/constant"
)

func newAsset() (a *src.Asset) {
  a = &src.Asset{
    Symbol: "Foo",
    O: make([]float64, constant.WINDOW_SIZE),
    H: make([]float64, constant.WINDOW_SIZE),
    L: make([]float64, constant.WINDOW_SIZE),
    C: make([]float64, constant.WINDOW_SIZE),
  }
  return
}
