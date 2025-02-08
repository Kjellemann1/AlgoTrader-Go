
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


// Moves each element one step to the left, and inserts the new value at the tail.
func rollFloat(arr *[]float64, v float64) {
  copy((*arr)[:constant.WINDOW_SIZE-1], (*arr)[1:])
  (*arr)[constant.WINDOW_SIZE-1] = v
}


func (a *Asset) fillMissingMinutes(t time.Time) {
  if a.Time.IsZero() {
    return
  }
  missingMinutes := int(t.Sub(a.Time).Minutes()) -1
  if missingMinutes > 0 {
    for i := 0; i < missingMinutes; i++ {
      if a.lastCloseIsTrade {
        a.C[constant.WINDOW_SIZE-1] = a.C[constant.WINDOW_SIZE-2]
      } else {
        rollFloat(&a.C, a.C[constant.WINDOW_SIZE-1])
      }
      rollFloat(&a.O, a.O[constant.WINDOW_SIZE-1])
      rollFloat(&a.H, a.H[constant.WINDOW_SIZE-1])
      rollFloat(&a.L, a.L[constant.WINDOW_SIZE-1])
      a.lastCloseIsTrade = false
    }
  }
}


type strategyFunc func(*Asset)


type Asset struct {
  Symbol            string
  Positions         map[string]*Position
  AssetQty          decimal.Decimal
  AssetClass        string
  Time              time.Time
  ReceivedTime      time.Time
  lastCloseIsTrade  bool

  O                 []float64
  H                 []float64
  L                 []float64
  C                 []float64

  strategies        []strategyFunc
  channels          []chan struct{}

  Rwm               sync.RWMutex
  Mutex             sync.Mutex
}


func NewAsset(asset_class string, symbol string) (a *Asset) {
  a = &Asset{
    lastCloseIsTrade: false,
    Positions: make(map[string]*Position),
    AssetClass: asset_class,
    Symbol: symbol,
    AssetQty: decimal.NewFromInt(0),
    O: make([]float64, constant.WINDOW_SIZE),
    H: make([]float64, constant.WINDOW_SIZE),
    L: make([]float64, constant.WINDOW_SIZE),
    C: make([]float64, constant.WINDOW_SIZE),
    strategies: []strategyFunc{
      (*Asset).testCool,
      // (*Asset).testRSI,
      // (*Asset).testSMA,
      // (*Asset).testBBands,
      // (*Asset).testMomentum,
      // (*Asset).testRSI1,
      // (*Asset).testRSI2,
      // (*Asset).testRSI3,
    },
  }
  a.StartStrategies()
  return
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


func (a *Asset) UpdateWindowOnBar(o float64, h float64, l float64, c float64, t time.Time, received_time time.Time) {
  a.Rwm.Lock()
  defer a.Rwm.Unlock()
  a.fillMissingMinutes(t)
  if a.lastCloseIsTrade {
    a.C[constant.WINDOW_SIZE-1] = c
  } else {
    rollFloat(&a.C, c)
    a.Time = t
  }
  rollFloat(&a.O, o)
  rollFloat(&a.H, h)
  rollFloat(&a.L, l)
  a.Time = t
  a.ReceivedTime = received_time
  a.lastCloseIsTrade = false
}


func (a *Asset) UpdateWindowOnTrade(c float64, t time.Time, received_time time.Time) {
  a.Rwm.Lock()
  defer a.Rwm.Unlock()
  if a.lastCloseIsTrade {
    a.C[constant.WINDOW_SIZE - 1] = c
  } else {
    rollFloat(&a.C, c)
  }
  a.Time = t
  a.ReceivedTime = received_time
  a.lastCloseIsTrade = true
}


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


func (a *Asset) createPositionID(strat_name string) string {
  t := a.Time.Format(time.DateTime)
  position_id := "symbol[" + a.Symbol + "]_strat[" + strat_name + "]_time[" + t + "]"
  return position_id
}


func (a *Asset) initiatePositionObject(strat_name string, order_type string, side string, order_id string, trigger_time time.Time) {
  a.Rwm.Lock()
  a.Positions[strat_name] = NewPosition(a.Symbol)
  a.Rwm.Unlock()
  a.Mutex.Lock()
  defer a.Mutex.Unlock()
  if a.Positions[strat_name] == nil {
    log.Fatal("Position object is nil for symbol: ", a.Symbol)
  }
  pos := a.Positions[strat_name]
  pos.Rwm.Lock()
  defer pos.Rwm.Unlock()
  pos.Symbol = a.Symbol
  pos.AssetClass = a.AssetClass
  pos.StratName = strat_name
  pos.PositionID = order_id
  pos.OpenSide = side
  pos.OpenOrderType = order_type
  pos.OpenTriggerPrice = a.C[constant.WINDOW_SIZE-1]
  pos.OpenTriggerTime = trigger_time
  pos.OpenPriceTime = a.Time
  pos.OpenPriceReceivedTime = a.ReceivedTime
}


func (a *Asset) RemovePosition(strat_name string) {
  a.Rwm.Lock()
  defer a.Rwm.Unlock()
  delete(a.Positions, strat_name)
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


// Methods below this point are for being called from strategy functions


func (a *Asset) IndexSingle(arr *[]float64, i int) (num float64) {
  num = (*arr)[constant.WINDOW_SIZE - 1 - i]
  return
}


func (a *Asset) IndexArray(arr *[]float64, from int, to int) (slice []float64) {
  slice = (*arr)[(constant.WINDOW_SIZE - 1 - to):(constant.WINDOW_SIZE - from)]
  return
}


func (a *Asset) Open(side string, order_type string, strat_name string) {
  trigger_time := time.Now().UTC()
  if _, ok := a.Positions[strat_name]; ok {
    return
  }
  if a.ReceivedTime.Sub(a.Time) > constant.MAX_RECEIVED_TIME_DIFF_MS {
    log.Println("[ INFO ]\tOpen cancelled due received time diff too large", a.Symbol)
    return
  } 
  if trigger_time.Sub(a.Time) > constant.MAX_TRIGGER_TIME_DIFF_MS {
    log.Println("[ INFO ]\tOpen cancelled due trigger time diff too large", a.Symbol)
    return
  }
  if NNP.Flag == true {
    return
  }
  last_close := a.C[constant.WINDOW_SIZE-1]
  symbol := a.Symbol
  order_id := a.createPositionID(strat_name)
  a.Mutex.Unlock()
  a.initiatePositionObject(strat_name, order_type, side, order_id, trigger_time)
  err := a.sendOpenOrder(order_type, order_id, symbol, last_close)
  if err != nil {
    handlelog.Error(err, "Symbol", symbol, "Strat", strat_name, "OrderType", order_type, "Side", side)
    a.RemovePosition(strat_name)
    return
  }
  a.Mutex.Lock()
}


func (a *Asset) Close(order_type string, strat_name string) {
  trigger_time := time.Now().UTC()
  if _, ok := a.Positions[strat_name]; !ok {
    return
  }
  pos := a.Positions[strat_name]
  pos.Rwm.Lock()
  if pos.CloseOrderPending || pos.OpenOrderPending {
    return
  }
  open_side := pos.OpenSide
  symbol := pos.Symbol
  qty := pos.Qty
  order_id := pos.PositionID
  pos.CloseOrderPending = true
  pos.CloseTriggerTime = trigger_time
  pos.CloseOrderType = order_type
  pos.CloseTriggerPrice = a.C[constant.WINDOW_SIZE-1]
  pos.ClosePriceTime = a.Time
  pos.ClosePriceReceivedTime = a.ReceivedTime
  pos.Rwm.Unlock()
  err := a.sendCloseOrder(open_side, order_type, order_id, symbol, qty)
  if err != nil {
    handlelog.Error(err, symbol, order_id)
    return
  }
}


func (a *Asset) StopLoss(percent float64, strat_name string) {
  // TODO: Log if StopLoss triggered to db
  if _, ok := a.Positions[strat_name]; !ok {
    return
  }
  fill_price := (*a).Positions[strat_name].OpenFilledAvgPrice
  if fill_price == 0 {
    return
  }
  dev := (fill_price / a.IndexSingle(&a.C, 0) - 1) * 100
  if dev < (percent * -1) {
    a.Close("IOC", strat_name)
    log.Printf("[ INFO ]\tStopLoss\t%s\t%s", a.Symbol, strat_name)
  }
}


func (a *Asset) TakeProfit(percent float64, strat_name string) {
  // TODO: Log if Take Profit triggered to db
  if _, ok := a.Positions[strat_name]; !ok {
    return
  }
  fill_price := (*a).Positions[strat_name].OpenFilledAvgPrice
  if fill_price == 0 {
    return
  }
  dev := ( fill_price / a.IndexSingle(&a.C, 0) - 1) * 100
  if dev > percent {
    a.Close("IOC", strat_name)
    log.Printf("[ INFO ]\tTakeProfit\t%s\t%s", a.Symbol, strat_name)
  }
}


func (a *Asset) TrailingStop() {
  panic("TrailingStop not implemented")
}
