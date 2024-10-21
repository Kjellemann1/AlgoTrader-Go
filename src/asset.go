
package src

import (
  "fmt"
  "log"
  "time"
  "github.com/shopspring/decimal"
  "github.com/Kjellemann1/AlgoTrader-Go/src/order"
  "github.com/Kjellemann1/AlgoTrader-Go/src/util/push"
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


// Check if the sum of the position quantities is equal to the asset quantity
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
    AssetQty: decimal.NewFromInt(0),
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


func (a *Asset) CreateOrderID(action string, strat_name string) string {
  t, err := time.Parse(time.RFC3339, a.Time)
  if err != nil {
    log.Printf(
      "[ FATAL ]\tFailed to parse time in strategy.CreateOrderID()\n" +
      "  -> Closing all positions and shutting down\n" +
      "  -> Action: %s\n" +
      "  -> Time: %s\n" +
      "  -> Error: %s\n",
    action, a.Time, err)
    push.Error("Error parsing time in CreateOrderID()", err)
  }
  order_id := fmt.Sprintf(
    "action[%s]_symbol[%s]_strat[%s]_time[%s]",
    action, a.Symbol, strat_name, t.Format(time.DateTime),
  )
  return order_id
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
  rwmu.Lock()
  defer rwmu.Unlock()
  // Check if position already exists
  if _, ok := a.Positions[strat_name]; ok {
    return
  }
  // Initiate position object
  a.Positions[strat_name] = &Position{
    OpenOrderPending: true,
    OpenTriggerTime: time.Now().UTC(),
  }
  // Send order
  var err error
  order_id := a.CreateOrderID("open", strat_name)
  switch order_type {
    case "IOC":
      err = order.OpenLongIOC(a.Symbol, order_id, a.Close[WINDOW_SIZE-1])
  }
  if err != nil {
    log.Printf(
      "[ ERROR ]\tFailed to open long position in OpenLongIOC()\n" +
      "  -> Symbol: %s\n" +
      "  -> Order ID: %s\n" +
      "  -> Error: %s\n",
    a.Symbol, order_id, err)
    push.Error("Error opening long position in OpenLongIOC()", err)
    delete(a.Positions, strat_name)
    return
  }
  order_sent_time := time.Now().UTC()
  // Fill out the rest of the position object
  pos := a.Positions[strat_name]
  pos.OpenOrderSentTime = order_sent_time
  pos.Symbol = a.Symbol
  pos.AssetClass = a.AssetClass
  pos.StratName = strat_name
  pos.Qty, _ = decimal.NewFromString("0")
  pos.BadForAnalysis = false
  pos.PositionID = order_id
  pos.OpenSide = side
  pos.OpenOrderType = order_type
  pos.OpenTriggerPrice = a.Close[WINDOW_SIZE-1]
  pos.OpenPriceTime = a.Time
}


func (a *Asset) ClosePosition(side string, order_type string, strat_name string) {
  // TODO: None of this is correct, just placeholder
  rwmu.Lock()
  defer rwmu.Unlock()
  // Check if position already exists
  if _, ok := a.Positions[strat_name]; ok {
    return
  }
  // Initiate position object
  a.Positions[strat_name] = &Position{
    OpenOrderPending: true,
    OpenTriggerTime: time.Now().UTC(),
  }
  // Send order
  var err error
  order_id := a.CreateOrderID("open", strat_name)
  switch order_type {
    case "IOC":
      err = order.OpenLongIOC(a.Symbol, order_id, a.Close[WINDOW_SIZE-1])
  }
  if err != nil {
    log.Printf(
      "[ ERROR ]\tFailed to open long position in OpenLongIOC()\n" +
      "  -> Symbol: %s\n" +
      "  -> Order ID: %s\n" +
      "  -> Error: %s\n",
    a.Symbol, order_id, err)
    push.Error("Error opening long position in OpenLongIOC()", err)
    delete(a.Positions, strat_name)
    return
  }
  // Fill out the rest of the position object
  pos := a.Positions[strat_name]
  pos.OpenOrderSentTime = time.Now().UTC()
  pos.Symbol = a.Symbol
  pos.AssetClass = a.AssetClass
  pos.StratName = strat_name
  pos.Qty, _ = decimal.NewFromString("0")
  pos.BadForAnalysis = false
  pos.PositionID = order_id
  pos.OpenSide = side
  pos.OpenOrderSentTime = time.Now().UTC()
  pos.OpenOrderType = order_type
  pos.OpenTriggerPrice = a.Close[WINDOW_SIZE-1]
  pos.OpenPriceTime = a.Time
}


func (s *Asset) CheckForSignal() {
  s.testingStrategy()
}
