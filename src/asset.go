
package src

import (
  "time"
  "github.com/shopspring/decimal"
)

const BUFFER_SIZE int = 100

type Asset struct {
  TotalQty         decimal.Decimal
  Open             [BUFFER_SIZE]float64
  High             [BUFFER_SIZE]float64
  Low              [BUFFER_SIZE]float64
  Close            [BUFFER_SIZE]float64
  Time             time.Time
  bar_updated_flag bool
}

func NewAsset() *Asset {
  return &Asset{
    TotalQty: decimal.NewFromFloat(0),
    bar_updated_flag: false,
  }
}

// Rolls the buffer, moving each element one step to the right, and inserting the new value at the first position.
func rollInt(arr *[BUFFER_SIZE]int, v int) {
  for i := len(arr) - 1; i > 0; i-- {
    arr[i] = arr[i-1]
  }
  arr[0] = v
}

func rollFloat(arr *[BUFFER_SIZE]float64, v float64) {
  for i := len(arr) - 1; i > 0; i-- {
    arr[i] = arr[i-1]
  }
  arr[0] = v
}

func rollTime(arr *[BUFFER_SIZE]time.Time, v time.Time) {
  for i := len(arr) - 1; i > 0; i-- {
    arr[i] = arr[i-1]
  }
  arr[0] = v
}

func (a *Asset) MarketUpdateBar(o float64, h float64, l float64, c float64, t time.Time) {
  rollFloat(&a.Open, o)
  rollFloat(&a.High, h)
  rollFloat(&a.Low, l)
  rollFloat(&a.Close, c)
  a.Time = t
}

func (a *Asset) MarketUpdateTrade() {
  if a.bar_updated_flag {

  }
}
