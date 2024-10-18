
package src

import (
  // External packages
  "github.com/shopspring/decimal" // https://pkg.go.dev/github.com/shopspring/decimal#section-readme
)

const WINDOW_SIZE int = 1000

// Moves each element one step to the left, and inserts the new value at the last position.
func rollInt(arr *[WINDOW_SIZE]int, v int) {
  copy(arr[:len(arr)-1], arr[1:])
  arr[len(arr)-1] = v
}

func rollFloat(arr *[WINDOW_SIZE]float64, v float64) {
  copy(arr[:len(arr)-1], arr[1:])
  arr[len(arr)-1] = v
}


// Asset struct
type Asset struct {
  Symbol           string
  TotalQty         decimal.Decimal
  Open             [WINDOW_SIZE]float64  // For rolling windows alone a linked list would be more efficient. However, the
  High             [WINDOW_SIZE]float64  // window is used for calculations that are repeated on every trade/bar update,
  Low              [WINDOW_SIZE]float64  // meaning they are not necessarily more efficient as a whole, and would significantly
  Close            [WINDOW_SIZE]float64  // add to the complexity of calculating indicators.
  Time             string                // No need to convert Time to time.Time until necessary
  lastCloseIsTrade bool
}


// Constructor for Asset
func NewAsset() *Asset {
  return &Asset{
    lastCloseIsTrade: false,
  }
}


// Updates the window on Bar updates
func (a *Asset) UpdateWindowOnBar(o float64, h float64, l float64, c float64, t string) {
  rwmu.Lock()
  defer rwmu.Unlock()
  if a.lastCloseIsTrade {
    a.Close[WINDOW_SIZE-1] = c
  } else {
    rollFloat(&a.Close, c)
    a.Time = t
  }
  rollFloat(&a.Open, o)
  rollFloat(&a.High, h)
  rollFloat(&a.Low, l)
  a.Time = t
  a.lastCloseIsTrade = false
}


// Updates the windows on Trade updates
// This is not trade updates from the algo, but from other trades in the market
func (a *Asset) UpdateWindowOnTrade(c float64, t string) {
  rwmu.Lock()
  defer rwmu.Unlock()
  if a.lastCloseIsTrade {
    a.Close[WINDOW_SIZE - 1] = c
  } else {
    rollFloat(&a.Close, c)
  }
  a.Time = t
  a.lastCloseIsTrade = true
}
