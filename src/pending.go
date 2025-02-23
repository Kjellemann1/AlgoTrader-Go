// When a reconnect is triggered in account, there could be missed order updates.
// Therefore we need to check for any orders that where executed while the connection
// was down, and update the positions accordingly.

package main

import(
  "fmt"
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
  float, err := strconv.ParseFloat(string(byte), 64)
  if err != nil {
    util.Warning(errors.New("failed to convert filled_avg_price to float in order update"))
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
    util.Warning(errors.New("failed to convert fill_time to time.Time in update"))
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

func (a *Account) parseClosedOrders(relevant map[string][]*fastjson.Value) map[string][]*ParsedClosedOrder {
  parsed := make(map[string][]*ParsedClosedOrder)
  for symbol, arr := range relevant {
    for _, fjv := range arr {
      co := &ClosedOrder{fjv}
      parsed[symbol] = append(parsed[symbol], co.parse())
    }
  }
  return parsed
}

func (a *Account) sendCloseGTC(diff decimal.Decimal, symbol string) {
  backoff_sec := 1
  retries := 0
  for {
    status, err := request.CloseGTC("sell", symbol, "strat[reconnect_mutliple_diff]", diff)
    if err != nil {
      util.Error(err, "Failed to send close order", "...")
    }
    switch status {
    case 403:
      util.Warning(errors.New("forbidden Block"),
        "Retries", retries,
        "Trying again in (seconds)", &backoff_sec,
      )
      util.Backoff(&backoff_sec)
    case 200:
      if retries > 0 {
        util.Warning(errors.New("order sent after retries"), "Retries", retries)
      }
      log.Println("[ OK ]\tReconciliation close order sent")
      return
    default:
      util.Error(fmt.Errorf("failed to send close order. Status: %d", status),
        "Retries", retries,
        "Trying again in (seconds)", &backoff_sec,
      )
      util.Backoff(&backoff_sec)
    }
    if retries >= 5 {
      util.Error(errors.New("max retries reached"),
        "CLOSING ALL POSITIONS AND SHUTTING DOWN", "...",
      )
      request.CloseAllPositions(2, 0)
      log.Panicln("SHUTTING DOWN")
    }
    retries++
  }
}

func (a *Account) diffMultiple(parsed []*ParsedClosedOrder, asset_class string) {
  asset := a.assets[asset_class][*parsed[0].Symbol]
  for _, pco := range parsed {
    pos := asset.Positions[*pco.StratName]
    pos.BadForAnalysis = true
    pos.Qty = decimal.Zero
    a.db_chan <-pos.LogClose()
    asset.removePosition(*pco.StratName)
  }
  diff_no_pending := asset.Qty.Sub(asset.sumNoPendingPosQtys())
  if !diff_no_pending.IsZero() {
    a.sendCloseGTC(diff_no_pending, *parsed[0].Symbol)
  }
}

func (a *Account) diffPositive(diff decimal.Decimal, asset_class string, parsed []*ParsedClosedOrder) {
  pco := parsed[0]
  asset := a.assets[asset_class][*pco.Symbol]
  pos := asset.Positions[*pco.StratName]
  pos.BadForAnalysis = true
  pos.Qty = diff
  pos.OpenFilledAvgPrice = *pco.FilledAvgPrice
  pos.OpenFillTime = *pco.FillTime
  pos.OpenOrderPending = false
}

func (a *Account) diffNegative(diff decimal.Decimal, asset_class string, parsed []*ParsedClosedOrder) {
  pco := parsed[0]
  asset := a.assets[asset_class][*pco.Symbol]
  pos := asset.Positions[*pco.StratName]
  pos.BadForAnalysis = true
  if diff.LessThan(pos.Qty) {
    pos.Qty = diff
    pos.NCloseOrders++
    pos.CloseOrderPending = false
    asset.close("IOC", pos.StratName)
    return
  }
  pos.Qty = decimal.Zero
  pos.CloseFilledAvgPrice = *pco.FilledAvgPrice
  pos.CloseFillTime = *pco.FillTime
  a.db_chan <-pos.LogClose()
  asset.removePosition(*pco.StratName)
}

func (a *Account) diffZero(asset_class string, parsed []*ParsedClosedOrder) {
  pco := parsed[0]
  asset := a.assets[asset_class][*pco.Symbol]
  pos := asset.Positions[*pco.StratName]
  if *pco.Side == "buy" {
    asset.removePosition(*pco.StratName)
    return
  }
  pos.BadForAnalysis = true
  pos.NCloseOrders++
  pos.CloseOrderPending = false
  asset.close("IOC", pos.StratName)
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
      if len(parsed[symbol]) > 1 {
        a.diffMultiple(parsed[symbol], asset_class)
        continue
      }
      diff := qtys[symbol].Sub((*a.assets[asset_class][symbol]).Qty)
      a.assets[asset_class][symbol].Qty = qtys[symbol]
      switch {
      case diff.IsPositive():
        a.diffPositive(diff, asset_class, parsed[symbol])
      case diff.IsNegative():
        a.diffNegative(diff, asset_class, parsed[symbol])
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
          util.Warning(errors.New("symbol not found"), nil)
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
  parsed := a.parseClosedOrders(relevant)
  a.updatePositions(parsed)
  log.Println("[ OK ]\tPending orders updated")
}
