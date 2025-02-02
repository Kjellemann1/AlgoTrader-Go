
package src

import (
  "log"
  "time"
  "sync"
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

type strategyFunc func(*Asset)

type Asset struct {
  Symbol           string
  Positions        map[string]*Position
  AssetQty         decimal.Decimal
  AssetClass       string
  Time             time.Time
  ReceivedTime     time.Time
  ProcessTime      time.Time
  lastCloseIsTrade bool

  Open             [constant.WINDOW_SIZE]float64
  High             [constant.WINDOW_SIZE]float64
  Low              [constant.WINDOW_SIZE]float64
  Close            [constant.WINDOW_SIZE]float64

  strategies      []strategyFunc
  channels        []chan struct{}

  rwm              sync.RWMutex
  mutex            sync.Mutex
}

func (a *Asset) StartStrategies() {
  n := len(a.strategies)
  a.channels = make([]chan struct{}, n)
  for i := 0; i < n; i++ {
    a.channels[i] = make(chan struct{})
    go func(idx int) {
      for range a.channels[idx] {
        a.strategies[idx](a)
      }
    }(i)
  }
}


func (a *Asset) CheckForSignal() {
  for i := range a.channels {
    if i >= 0 && i < len(a.channels) {
      a.channels[i] <- struct{}{}
    }
  }
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
func NewAsset(asset_class string, symbol string) (a *Asset) {
  a = &Asset{
    lastCloseIsTrade: false,
    Positions: make(map[string]*Position),
    AssetClass: asset_class,
    Symbol: symbol,
    AssetQty: decimal.NewFromInt(0),
    strategies: []strategyFunc{
      (*Asset).testingRand1,
      (*Asset).testingRand2,
    },
  }
  a.StartStrategies()
  return
}




// Updates the window on Bar updates
func (a *Asset) UpdateWindowOnBar(
  o float64, h float64, l float64, c float64, t time.Time, received_time time.Time, process_time time.Time,
) {
  a.rwm.Lock()
  defer a.rwm.Unlock()
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
  a.ProcessTime = process_time
  a.lastCloseIsTrade = false
}


// Updates the windows on Trade updates
func (a *Asset) UpdateWindowOnTrade(c float64, t time.Time, received_time time.Time, process_time time.Time) {
  a.rwm.Lock()
  defer a.rwm.Unlock()
  if a.lastCloseIsTrade {
    a.Close[constant.WINDOW_SIZE - 1] = c
  } else {
    rollFloat(&a.Close, c)
  }
  a.Time = t
  a.ReceivedTime = received_time
  a.ProcessTime = process_time
  a.lastCloseIsTrade = true
}


func (a *Asset) RemovePosition(strat_name string) {
  a.rwm.Lock()
  defer a.rwm.Unlock()
  delete(a.Positions, strat_name)
}


func (a *Asset) createPositionID(strat_name string) string {
  t := a.Time.Format(time.DateTime)
  position_id := "symbol[" + a.Symbol + "]_strat[" + strat_name + "]_time[" + t + "]"
  return position_id
}


func (a *Asset) sendOpenOrder(order_type string, order_id string, symbol string, last_close float64) (err error) {
  switch order_type {
    case "IOC":
      err = order.OpenLongIOC(symbol, order_id, last_close)
  }
  return
}


func (a *Asset) sendCloseOrder(open_side, order_type string, order_id string, symbol string, qty decimal.Decimal) (err error) {
  var side string
  switch open_side {
  case "long":
    side = "sell"
  case "short":
    side = "buy"
  }
  switch order_type {
    case "IOC":
      err = order.CloseIOC(side, symbol, order_id, qty)
  }
  return
}


func (a *Asset) initiatePositionObject(strat_name string, order_type string, side string, order_id string, trigger_time time.Time) {
  a.rwm.Lock()
  a.Positions[strat_name] = NewPosition(a.Symbol)
  a.rwm.Unlock()
  a.mutex.Lock()
  defer a.mutex.Unlock()
  if a.Positions[strat_name] == nil {
    log.Fatal("Position object is nil for symbol: ", a.Symbol)
  }
  pos := a.Positions[strat_name]
  pos.rwm.Lock()
  defer pos.rwm.Unlock()
  pos.Symbol = a.Symbol
  pos.AssetClass = a.AssetClass
  pos.StratName = strat_name
  pos.Qty, _ = decimal.NewFromString("0")
  pos.PositionID = order_id
  pos.OpenSide = side
  pos.OpenOrderType = order_type
  pos.OpenTriggerPrice = a.Close[constant.WINDOW_SIZE-1]
  pos.OpenTriggerTime = trigger_time
  pos.OpenPriceTime = a.Time
  pos.OpenPriceReceivedTime = a.ReceivedTime
  pos.OpenPriceProcessTime = a.ProcessTime
}


func (a *Asset) OpenPosition(side string, order_type string, strat_name string) {
  if _, ok := a.Positions[strat_name]; ok {
    return
  }
  if a.ReceivedTime.Sub(a.Time) > constant.MAX_RECEIVED_TIME_DIFF_MS {
    log.Println("[ INFO ]\tOpen cancelled due to time diff too large", a.Symbol)
    return
  }
  trigger_time := time.Now().UTC()
  last_close := a.Close[constant.WINDOW_SIZE-1]
  symbol := a.Symbol
  order_id := a.createPositionID(strat_name)
  a.mutex.Unlock()
  a.initiatePositionObject(strat_name, order_type, side, order_id, trigger_time)
  err := a.sendOpenOrder(order_type, order_id, symbol, last_close)
  if err != nil {
    handlelog.Error(err, "Symbol", symbol, "Strat", strat_name, "OrderType", order_type, "Side", side)
    a.RemovePosition(strat_name)
    return
  }
  a.mutex.Lock()
}


func (a *Asset) ClosePosition(order_type string, strat_name string) {
  if _, ok := a.Positions[strat_name]; !ok {
    return
  }
  pos := a.Positions[strat_name]
  if pos.CloseOrderPending || pos.OpenOrderPending {
    return
  }
  trigger_time := time.Now().UTC()
  open_side := pos.OpenSide
  symbol := pos.Symbol
  qty := pos.Qty
  order_id := pos.PositionID
  pos.rwm.Lock()
  pos.CloseOrderPending = true
  pos.CloseTriggerTime = trigger_time
  pos.CloseOrderType = order_type
  pos.CloseTriggerPrice = a.Close[constant.WINDOW_SIZE-1]
  pos.ClosePriceTime = a.Time
  pos.ClosePriceReceivedTime = a.ReceivedTime
  pos.ClosePriceProcessTime = a.ProcessTime
  pos.rwm.Unlock()
  err := a.sendCloseOrder(open_side, order_type, order_id, symbol, qty)
  if err != nil {
    handlelog.Error(err, symbol, order_id)
    return
  }
}
