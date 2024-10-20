
package src

import (
  "fmt"
  "time"
  "log"
  "github.com/shopspring/decimal"

  "github.com/Kjellemann1/AlgoTrader-Go/src/order"
  "github.com/Kjellemann1/AlgoTrader-Go/src/util/push"
)

type Strategy struct {
  *Asset
}

func (s *Strategy) ClosePosition(strat_name string) {
  // TODO
}

// TODO: Make suer time.DateTime has not been used as layout
func (s *Strategy) CreateOrderID(action string, strat_name string) string {
  t, err := time.Parse(time.RFC3339, s.Time)
  if err != nil {
    log.Printf(
      "[ FATAL ]\tFailed to parse time in strategy.CreateOrderID()\n" +
      "  -> Closing all positions and shutting down\n" +
      "  -> Action: %s\n" +
      "  -> Time: %s\n" +
      "  -> Error: %s\n",
    action, s.Time, err)
    push.Error("Error parsing time in CreateOrderID()", err)
  }
  order_id := fmt.Sprintf(
    "action[%s]_symbol[%s]_strat[%s]_time[%s]",
    action, s.Symbol, strat_name, t.Format(time.DateTime),
  )
  return order_id
}

func (s *Strategy) InitiatePositionObject(strat_name string) {
  p := &Position {
    Symbol: s.Symbol,
    AssetClass: s.AssetClass,
    StratName: strat_name,
    Qty: decimal.NewFromInt(0),
    BadForAnalysis: false,
    OpenTriggerTime: time.Now().UTC(),
  }
  s.Positions[strat_name] = p
}


func (s *Strategy) Open(side string, order_type string, strat_name string) {
  rwmu.Lock()
  defer rwmu.Unlock()
  // Check if position already exists
  if _, ok := s.Positions[strat_name]; ok {
    return
  }
  // Initiate position object
  s.Positions[strat_name] = &Position{
    OpenOrderPending: true,
    OpenTriggerTime: time.Now().UTC(),
  }
  // Open position
  var err error
  order_id := s.CreateOrderID("open", strat_name)
  switch order_type {
    case "IOC":
      err = order.OpenLongIOC(s.Symbol, order_id, s.Close[WINDOW_SIZE-1])
  }
  if err != nil {
    log.Printf(
      "[ ERROR ]\tFailed to open long position in OpenLongIOC()\n" +
      "  -> Symbol: %s\n" +
      "  -> Order ID: %s\n" +
      "  -> Error: %s\n",
    s.Symbol, order_id, err)
    push.Error("Error opening long position in OpenLongIOC()", err)
    delete(s.Positions, strat_name)
    return
  }
  // Update position object
  pos := s.Positions[strat_name]
  pos.Symbol = s.Symbol
  pos.AssetClass = s.AssetClass
  pos.StratName = strat_name
  pos.Qty, _ = decimal.NewFromString("0")
  pos.BadForAnalysis = false
  pos.OrderID = order_id
  pos.OpenSide = side
  pos.OpenOrderSentTime = time.Now().UTC()
  pos.OpenOrderType = order_type
  pos.OpenTriggerPrice = s.Close[WINDOW_SIZE-1]
  pos.OpenPriceTime = s.Time
}


func (s *Strategy) CheckForSignal() {
  s.TestingStrategy()
}

var testCount int = 0


func (s *Strategy) TestingStrategy() {
  if testCount >= 3 {
    return
  }
  s.Open("long", "IOC", "test")
  testCount++
}
