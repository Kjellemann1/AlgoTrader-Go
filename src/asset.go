
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
  Positions        map[string]*Position
  AssetQty         decimal.Decimal
  AssetClass       string
  Open             [WINDOW_SIZE]float64
  High             [WINDOW_SIZE]float64
  Low              [WINDOW_SIZE]float64
  Close            [WINDOW_SIZE]float64
  Time             string
  lastCloseIsTrade bool
}


func (a *Asset) SumPosQtyEqAssetQty() bool {
  rwmu.RLock()
  defer rwmu.RUnlock()
  count, _ := decimal.NewFromString("0")
  for _, val := range a.Positions {
    count = count.Add(val.Qty)
  }
  if a.AssetQty.Compare(count) != 0 {
    return false
  } else {
    return true
  }
}


// Constructor for Asset
func NewAsset(asset_class string, symbol string) *Asset {
  return &Asset{
    lastCloseIsTrade: false,
    Positions: make(map[string]*Position),
    AssetClass: asset_class,
    Symbol: symbol,
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


// Remove position
// TODO: Remove position from database as well
func (a *Asset) RemovePosition(strat_name string) {
  rwmu.Lock()
  defer rwmu.Unlock()
  delete(a.Positions, strat_name)
}
