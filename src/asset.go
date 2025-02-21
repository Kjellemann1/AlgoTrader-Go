package main

import (
  "log"
  "time"
  "sync"
  "errors"
  "github.com/shopspring/decimal"
  "github.com/Kjellemann1/AlgoTrader-Go/request"
  "github.com/Kjellemann1/AlgoTrader-Go/constant"
  "github.com/Kjellemann1/AlgoTrader-Go/util"
)

func prepAssetsMap() map[string]map[string]*Asset {
  assets := make(map[string]map[string]*Asset)

  if len(constant.STOCK_LIST) > 0 {
    assets["stock"] = make(map[string]*Asset)
    for _, symbol := range constant.STOCK_LIST {
      assets["stock"][symbol] = newAsset("stock", symbol)
    }
  }

  if len(constant.CRYPTO_LIST) > 0 {
    assets["crypto"] = make(map[string]*Asset)
    for _, symbol := range constant.CRYPTO_LIST {
      assets["crypto"][symbol] = newAsset("crypto", symbol)
    }
  }

  return assets
}

func pendingOrders(assets map[string]map[string]*Asset) map[string][]*Position {
  pending := make(map[string][]*Position, 0)

  for _, asset_class := range assets {
    for _, asset := range asset_class {
      for _, pos := range asset.Positions {
        if pos.OpenOrderPending || pos.CloseOrderPending {
          pending[pos.Symbol] = append(pending[pos.Symbol], pos)
        }
      }
    }
  }

  return pending
}

func positionsSymbols(positions map[string][]*Position) map[string]map[string]int {
  symbols := make(map[string]map[string]int)

  for s, l := range positions {
    for _, p := range l {
      if p.AssetClass == "stock" {
        symbols["stock"][s] = 1
      } else {
        symbols["crypto"][s] = 1
      }
    }
  }

  return symbols
}

// Moves each element one step to the left, and inserts the new value at the tail.
func rollFloat(arr *[]float64, v float64) {
  copy((*arr)[:constant.WINDOW_SIZE-1], (*arr)[1:])
  (*arr)[constant.WINDOW_SIZE-1] = v
}

func (a *Asset) fillMissingMinutes(t time.Time) {
  if a.Time.IsZero() {
    return
  }
  if a.AssetClass == "stock" {
    if a.Time.Day() != t.Day() {
      return
    }
  }
  missingMinutes := int(t.Sub(a.Time).Minutes()) - 1
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

func newAsset(asset_class string, symbol string) (a *Asset) {
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
      (*Asset).rand,
    },
  }
  a.startStrategies()
  return
}

func (a *Asset) startStrategies() {
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

func (a *Asset) checkForSignal() {
  for i := range a.channels {
    if i >= 0 && i < len(a.channels) {
      a.channels[i] <- struct{}{}
    }
  }
}

func (a *Asset) updateWindowOnBar(o float64, h float64, l float64, c float64, t time.Time, received_time time.Time) {
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

func (a *Asset) updateWindowOnTrade(c float64, t time.Time, received_time time.Time) {
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

func (a *Asset) sumPosQtyEqAssetQty() bool {
  count, _ := decimal.NewFromString("0")
  for _, val := range a.Positions {
    count = count.Add(val.Qty)
  }
  if a.AssetQty.Compare(count) != 0 {
    log.Printf(
      "[ INFO ]\tAssetQty does not equal sum of position quantities\n" +
      "  ->AssetQty: %s\n\tSum of position quantities: %s\n\tSymbol: %s\n",
    a.AssetQty.String(), count.String(), a.Symbol)
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
    util.Error(errors.New("Position object is nil for symbol: " + a.Symbol),
      "StratName", strat_name, "OrderType", order_type, "Side", side, "OrderID", order_id,
      "CLOSING ALL POSITIONS AND SHUTTING DOWN", "...",
    )
    request.CloseAllPositions(2, 0)
    panic("SHUTTING DOWN")
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

func (a *Asset) removePosition(strat_name string) {
  a.Rwm.Lock()
  defer a.Rwm.Unlock()
  delete(a.Positions, strat_name)
}

func (a *Asset) sendOpenOrder(order_type string, order_id string, symbol string, asset_class string, last_close float64) (err error) {
  switch order_type {
    case "IOC":
      err = request.OpenLongIOC(symbol, asset_class, order_id, last_close)
      if err != nil {
        return
      }
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
      err = request.CloseIOC(side, symbol, order_id, qty)
  }
  return
}

// Methods below this point are for being called from strategy functions
func (a *Asset) i(num int) (index int) {
	index = constant.WINDOW_SIZE - 1 - num
	return
}

func (a *Asset) s(arr *[]float64, from int, to int) (slice []float64) {
  slice = (*arr)[(constant.WINDOW_SIZE - 1 - to):(constant.WINDOW_SIZE - from)]
  return
}

func (a *Asset) open(side string, order_type string, strat_name string) {
  trigger_time := time.Now().UTC()

  if _, ok := a.Positions[strat_name]; ok {
    log.Println("[ INFO ]\tOpen cancelled since trade already exists", a.Symbol)
    return
  }

  if a.ReceivedTime.Sub(a.Time) > constant.MAX_RECEIVED_TIME_DIFF_MS {
    log.Println("[ INFO ]\tOpen cancelled due to received time diff too large", a.Symbol)
    return
  } 

  if trigger_time.Sub(a.Time) > constant.MAX_TRIGGER_TIME_DIFF_MS {
    log.Println("[ INFO ]\tOpen cancelled due to trigger time diff too large", a.Symbol)
    return
  }

  if NNP.Flag == true {
    return
  }

  last_close := a.C[constant.WINDOW_SIZE-1]
  symbol := a.Symbol
  order_id := a.createPositionID(strat_name)
  asset_class := a.AssetClass
  a.Mutex.Unlock()

  a.initiatePositionObject(strat_name, order_type, side, order_id, trigger_time)
  err := a.sendOpenOrder(order_type, order_id, symbol, asset_class, last_close)
  if err != nil {
    util.Warning(err, "Symbol", symbol, "Strat", strat_name, "OrderType", order_type, "Side", side)
    a.removePosition(strat_name)
  }

  a.Mutex.Lock()
}

func (a *Asset) close(order_type string, strat_name string) {
  trigger_time := time.Now().UTC()

  if _, ok := a.Positions[strat_name]; !ok {
    return
  }

  pos := a.Positions[strat_name]
  pos.Rwm.Lock()

  if pos.CloseOrderPending || pos.OpenOrderPending {
    log.Println("[ INFO ]\tClose cancelled due to order pending", a.Symbol)
    pos.Rwm.Unlock()
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
    util.Error(err, symbol, order_id)
    return
  }
}

func (a *Asset) stopLoss(percent float64, strat_name string) {
  // TODO:
  //   -> Log if stop loss triggered to db
  //   -> Implement logic for short positions
  if _, ok := a.Positions[strat_name]; !ok {
    return
  }

  fill_price := a.Positions[strat_name].OpenFilledAvgPrice
  if fill_price == 0 {
    return
  }

  dev := (fill_price / a.C[a.i(0)] - 1) * 100
  if dev < (percent * -1) {
    a.close("IOC", strat_name)
    log.Printf("[ INFO ]\tStopLoss\t%s\t%s", a.Symbol, strat_name)
  }
}

func (a *Asset) takeProfit(percent float64, strat_name string) {
  // TODO:
  //   -> Log if take profit triggered to db
  //   -> Implement logic for short positions
  if _, ok := a.Positions[strat_name]; !ok {
    return
  }

  fill_price := a.Positions[strat_name].OpenFilledAvgPrice
  if fill_price == 0 {
    return
  }

  dev := (fill_price / a.C[a.i(0)] - 1) * 100
  if dev > percent {
    a.close("IOC", strat_name)
    log.Printf("[ INFO ]\tTakeProfit\t%s\t%s", a.Symbol, strat_name)
  }
}

func (a *Asset) trailingStop() {
  // TODO
  log.Println("[ INFO ]\tTrailingStop not implemented")
  return
}
