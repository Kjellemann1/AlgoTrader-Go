
package src

import (
  "log"
  "time"
  "sync"
  "fmt"
  "github.com/shopspring/decimal"
  "github.com/Kjellemann1/AlgoTrader-Go/src/order"
  "github.com/Kjellemann1/AlgoTrader-Go/src/constant"
  "github.com/Kjellemann1/AlgoTrader-Go/src/util/handlelog"
)


// Moves each element one step to the left, and inserts the new value at the last position.
func rollInt(arr *[constant.WINDOW_SIZE]int, v int) {
  copy(arr[:len(arr)-1], arr[1:])
  arr[len(arr)-1] = v
}


func rollFloat(arr *[constant.WINDOW_SIZE]float64, v float64) {
  copy(arr[:len(arr)-1], arr[1:])
  arr[len(arr)-1] = v
}


// Asset struct
type Asset struct {
  Symbol           string
  Positions        map[string]*Position
  AssetQty         decimal.Decimal
  AssetClass       string
  Open             [constant.WINDOW_SIZE]float64
  High             [constant.WINDOW_SIZE]float64
  Low              [constant.WINDOW_SIZE]float64
  Close            [constant.WINDOW_SIZE]float64
  Time             time.Time
  ReceivedTime     time.Time
  lastCloseIsTrade bool
  mutex            sync.RWMutex
}


// Check if the sum of the position quantities is equal to the asset quantity
func (a *Asset) SumPosQtyEqAssetQty() bool {
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
    AssetQty: decimal.NewFromInt(0),
  }
}


// Updates the window on Bar updates
func (a *Asset) UpdateWindowOnBar(o float64, h float64, l float64, c float64, t time.Time, received_time time.Time) {
  a.mutex.Lock()
  defer a.mutex.Unlock()
  if a.lastCloseIsTrade {
    a.Close[constant.WINDOW_SIZE-1] = c
  } else {
    rollFloat(&a.Close, c)
    a.Time = t
  }
  rollFloat(&a.Open, o)
  rollFloat(&a.High, h)
  rollFloat(&a.Low, l)
  a.Time = t
  a.ReceivedTime = received_time
  a.lastCloseIsTrade = false
}


// Updates the windows on Trade updates
func (a *Asset) UpdateWindowOnTrade(c float64, t time.Time, received_time time.Time) {
  a.mutex.Lock()
  defer a.mutex.Unlock()
  if a.lastCloseIsTrade {
    a.Close[constant.WINDOW_SIZE - 1] = c
  } else {
    rollFloat(&a.Close, c)
  }
  a.Time = t
  a.ReceivedTime = received_time
  a.lastCloseIsTrade = true
}


// Remove position
// TODO: Remove position from database as well
func (a *Asset) RemovePosition(strat_name string) {
  a.mutex.Lock()
  defer a.mutex.Unlock()
  delete(a.Positions, strat_name)
}


func (a *Asset) CreatePositionID(strat_name string) string {
  t := a.Time.Format(time.DateTime)
  position_id := fmt.Sprintf(
    "symbol[%s]_strat[%s]_time[%s]",
    a.Symbol, strat_name, t,
  )
  return position_id
}


func (a *Asset) InitiatePositionObject(strat_name string) {
  p := &Position {
    Symbol: a.Symbol,
    AssetClass: a.AssetClass,
    StratName: strat_name,
    Qty: decimal.NewFromInt(0),
    BadForAnalysis: false,
    OpenTriggerTime: time.Now().UTC(),
  }
  a.Positions[strat_name] = p
}


func (a *Asset) OpenPosition(side string, order_type string, strat_name string) {
  if side != "long" {
    log.Fatal("[ FATAL ]\tOnly long positions are supported")  // TODO: Add support for short positions
  }
  a.mutex.Lock()
  defer a.mutex.Unlock()
  // Check if diff between price time and received time is too large
  if a.ReceivedTime.Sub(a.Time) > constant.MAX_RECEIVED_TIME_DIFF_MS {
    log.Println("[ INFO ]\tOpen cancelled due to time diff too large", a.Symbol)
    return
  }
  // Check if position already exists
  if _, ok := a.Positions[strat_name]; ok {
    return
  }
  // Initiate position object
  a.Positions[strat_name] = NewPosition(a.Symbol, a.Close[constant.WINDOW_SIZE-1])
  if a.Positions[strat_name] == nil {
    log.Fatal("Position object is nil for symbol: ", a.Symbol)
  }
  // Send order
  var err error
  order_id := a.CreatePositionID(strat_name)
  order_time_before := time.Now().UTC()
  switch order_type {
    case "IOC":
      err = order.OpenLongIOC(a.Symbol, order_id, a.Close[constant.WINDOW_SIZE-1])
  }
  if err != nil {
    handlelog.Error(err, a.Symbol, order_id)
    delete(a.Positions, strat_name)
    return
  }
  order_time_after := time.Now().UTC()  // Remove
  // Fill out the rest of the position object
  pos := a.Positions[strat_name]
  pos.OpenOrderTimeBefore = order_time_before
  pos.OpenOrderTimeAfter = order_time_after
  pos.Symbol = a.Symbol
  pos.AssetClass = a.AssetClass
  pos.StratName = strat_name
  pos.Qty, _ = decimal.NewFromString("0")
  pos.PositionID = order_id
  pos.OpenSide = side
  pos.OpenOrderType = order_type
  pos.OpenTriggerPrice = a.Close[constant.WINDOW_SIZE-1]
  pos.OpenPriceTime = a.Time
  pos.OpenPriceReceivedTime = a.ReceivedTime
}


func (a *Asset) ClosePosition(order_type string, strat_name string) {
  a.mutex.Lock()
  defer a.mutex.Unlock()
  // Check if position already exists
  if _, ok := a.Positions[strat_name]; !ok {
    return
  }
  pos := a.Positions[strat_name]
  if pos.CloseOrderPending || pos.OpenOrderPending {
    return
  }
  pos.CloseOrderPending = true
  pos.CloseTriggerTime = time.Now().UTC()
  // Send order
  var err error
  order_time_before := time.Now().UTC()  // Remove
  switch order_type {
    case "IOC":
      switch pos.OpenSide {
        case "long":
          err = order.CloseIOC("sell", a.Symbol, pos.PositionID, pos.Qty)
      }
  }
  if err != nil {
    handlelog.Error(err, a.Symbol, pos.PositionID)
    return
  }
  order_time_after := time.Now().UTC()
  pos.CloseOrderTimeBefore = order_time_before
  pos.CloseOrderTimeAfter = order_time_after
  pos.CloseOrderType = order_type
  pos.CloseTriggerPrice = a.Close[constant.WINDOW_SIZE-1]
  pos.ClosePriceTime = a.Time
  pos.ClosePriceReceivedTime = a.ReceivedTime
}

func (a *Asset) CheckForSignal() {
  a.mutex.Lock()
  defer a.mutex.Unlock()
  a.testingStrategy()
}
