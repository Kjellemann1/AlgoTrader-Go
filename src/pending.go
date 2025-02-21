// When a reconnect is triggered in account, there could be missed order updates.
// Therefore we need to check for any orders that where executed while the connection
// was down, and update the positions accordingly.
//
// A simple approach is used where the difference between the asset qty on the server,
// and the asset qty in the asset struct, is equaly distributed between the positions
// with pending orders.
//
// Due to this approach, these trader will be marked as "bad for analysis" in the
// database, as the qty likely differs from the actual.

package main

import(
  "log"
  "time"
  "errors"
  "strconv"
  "github.com/valyala/fastjson"
  "github.com/shopspring/decimal"
  "github.com/Kjellemann1/AlgoTrader-Go/request"
  "github.com/Kjellemann1/AlgoTrader-Go/util"
)

type ClosedOrder struct {
  *fastjson.Value
}

type ParsedClosedOrder struct {
  Side *string
  StratName *string
  Symbol *string
  FilledAvgPrice *float64
  FilledQty *decimal.Decimal
  FillTime *time.Time
}

func (co *ClosedOrder) getString(element string) *string {
  byte := co.GetStringBytes(element)
  if byte == nil {
    return nil
  }
  str := string(byte)

  return &str
}

func (co *ClosedOrder) getStratName() *string {
  byte := co.GetStringBytes("client_order_id")
  if byte == nil {
    return nil
  }
  str := string(byte)
  return grepStratName(&str)
}

func (co *ClosedOrder) getFloat(element string) *float64 {
  byte := co.GetStringBytes(element)
  if byte == nil {
    return nil
  }

  float, err := strconv.ParseFloat(string(byte), 8)
  if err != nil {
    util.Warning(errors.New("Failed to convert filled_avg_price to float in order update"))
  }

  return &float
}

func (co *ClosedOrder) getFilledQty() *decimal.Decimal {
  byte := co.GetStringBytes("filled_qty")
  if byte == nil {
    return nil
  }

  dec, err := decimal.NewFromString(string(byte))
  if err != nil {
    NNP.NoNewPositionsTrue("")
    util.Error(err, "CLOSING ALL POSITIONS AND SHUTTING DOWN", "...")
    request.CloseAllPositions(2, 0)
    log.Panicln("SHUTTING DOWN")
  }

  return &dec
}

func (co *ClosedOrder) getFillTime() *time.Time {
  fill_time := co.GetStringBytes("filled_at")
  if fill_time == nil {
    return nil
  }

  fill_time_t, err := time.Parse(time.RFC3339, string(fill_time))
  if err != nil {
    util.Warning(errors.New("Failed to convert fill_time to time.Time in update"))
  }

  return &fill_time_t
}

func (co *ClosedOrder) parse() *ParsedClosedOrder {
  return &ParsedClosedOrder{
    Symbol: co.getString("symbol"),
    Side: co.getString("side"),
    StratName: co.getStratName(),
    FilledAvgPrice: co.getFloat("filled_avg_price"),
    FilledQty: co.getFilledQty(),
    FillTime: co.getFillTime(),
  }
}

func (a *Account) closedOrderHandler(arr []*fastjson.Value) map[string][]*ParsedClosedOrder {
  parsed := make(map[string][]*ParsedClosedOrder)
  for _, m := range arr {
    co := &ClosedOrder{m}
    pco := co.parse()
    parsed[*pco.Symbol] = append(parsed[*pco.Symbol], pco)
  }
  return parsed
}

func (a *Account) diffZero(asset_class string, parsed []*ParsedClosedOrder) {
  for _, pco := range parsed {
    asset := a.assets[asset_class][*pco.Symbol]
    asset.Positions[*pco.StratName].BadForAnalysis = true
    asset.Positions[*pco.StratName].Qty = decimal.Zero
    asset.removePosition(*pco.StratName)
  }
}

func bestRoundingFunctionEverMadeInTheHistoryOfTheUniverseNoKapp(num decimal.Decimal, desimals int64) decimal.Decimal {
  ten, _ := decimal.NewFromString("10")
  nine := decimal.NewFromInt(desimals)
  big := num.Mul(ten.Pow(nine))
  rounded := big.Floor().Div(ten.Pow(nine))
  return rounded
}

func (a *Account) diffPositive(diff *decimal.Decimal, n_pending_open int, asset_class string, parsed []*ParsedClosedOrder) {
  for _, pco := range parsed {
    asset := a.assets[asset_class][*pco.Symbol]
    switch *pco.Side {
    case "buy":
    }
  }
}

func (a *Account) updatePositions(parsed map[string][]*ParsedClosedOrder) {
  qtys, err := request.GetAssetQtys()
  if err != nil {
    NNP.NoNewPositionsTrue("")
    util.Error(err, "CLOSING ALL POSITIONS AND SHUTTING DOWN", "...")
    request.CloseAllPositions(2, 0)
    log.Panicln("SHUTTING DOWN")
  }

  for asset_class := range a.assets {
    for symbol := range parsed {
      n_pending_open := 0

      for _, co := range parsed[symbol] {
        if *co.Side == "buy" {
          n_pending_open++
        }
      }

      diff := qtys[symbol].Sub((*a.assets[asset_class][symbol]).AssetQty)

      switch {
      case diff.IsPositive():
        continue
      case diff.IsNegative():
        continue
      default:
        a.diffZero(asset_class, parsed[symbol])
      }
    }
  }
}


func (a *Account) filterRelevantOrders(arr []*fastjson.Value, pending map[string][]*Position) map[string][]*fastjson.Value {
  relevant := make(map[string][]*fastjson.Value)

  for _, m := range arr {
    for _, v := range pending {
      for _, pos := range v {
        position_id := m.GetStringBytes("client_order_id")  // client_order_id == PositionID
        symbol := m.GetStringBytes("symbol")
        if position_id == nil {
          util.Warning(errors.New("PositionID not found"), nil)
          continue
        } else if symbol == nil {
          util.Warning(errors.New("Symbol not found"), nil)
          continue
        }
        if string(position_id) == pos.PositionID {
          relevant[string(symbol)] = append(relevant[string(symbol)], m)
          break
        }
      }
    }
  }

  return relevant
}

func (a *Account) parseOrder(order *ClosedOrder) {

}

func (a *Account) parseOrders() {

}

func (a *Account) checkPending() {
  globRwm.RLock()
  defer globRwm.RUnlock()

  pending := pendingOrders(a.assets)
  if len(pending) == 0 {
    log.Println("[ OK ]\tNo pending orders")
    return
  }

  arr, err := request.GetClosedOrders(positionsSymbols(pending), 5, 3)
  if err != nil {
    NNP.NoNewPositionsTrue("")
    util.Error(err, "CLOSING ALL POSITIONS AND SHUTTING DOWN", "...")
    request.CloseAllPositions(2, 0)
    log.Panicln("SHUTTING DOWN")
  }

  relevant := a.filterRelevantOrders(arr, pending)
  if len(relevant) == 0 {
    log.Println("[ OK ]\tNo pending orders closed")
    return
  }

  // a.checkAssetQty()

  log.Println("[ OK ]\tPending orders updated")
}
