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
  if len(constant.STOCK_SYMBOLS) > 0 {
    assets["stock"] = make(map[string]*Asset)
    for _, symbol := range constant.STOCK_SYMBOLS {
      assets["stock"][symbol] = newAsset("stock", symbol)
    }
  }
  if len(constant.CRYPTO_SYMBOLS) > 0 {
    assets["crypto"] = make(map[string]*Asset)
    for _, symbol := range constant.CRYPTO_SYMBOLS {
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
  for _, l := range positions {
    for _, p := range l {
      if _, ok := symbols[p.AssetClass]; !ok {
        symbols[p.AssetClass] = make(map[string]int)
      }
    }
  }
  for s, l := range positions {
    for _, p := range l {
      symbols[p.AssetClass][s] = 1
    }
  }
  return symbols
}

func (a *Asset) sumPosQtysEqAssetQty() bool {
  count, _ := decimal.NewFromString("0")
  for _, val := range a.Positions {
    count = count.Add(val.Qty)
  }
  if a.Qty.Compare(count) != 0 {
    return false
  } else {
    return true
  }
}

func (a *Asset) sumNoPendingPosQtys() decimal.Decimal {
  count, _ := decimal.NewFromString("0")
  for _, val := range a.Positions {
    if !val.OpenOrderPending && !val.CloseOrderPending {
      count = count.Add(val.Qty)
    }
  }
  return count
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
  if a.Class == "stock" {
    if a.Time.Day() != t.Day() {
      return
    }
  }
  missingMinutes := int(t.Sub(a.Time).Minutes()) - 1
  if missingMinutes > 0 {
    for range missingMinutes {
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
  Qty               decimal.Decimal
  Class             string
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

  close            func(string, string)
  open             func(string, string, string)
}

func newAsset(asset_class string, symbol string) (a *Asset) {
  a = &Asset{
    lastCloseIsTrade: false,
    Positions: make(map[string]*Position),
    Class: asset_class,
    Symbol: symbol,
    Qty: decimal.NewFromInt(0),
    O: make([]float64, constant.WINDOW_SIZE),
    H: make([]float64, constant.WINDOW_SIZE),
    L: make([]float64, constant.WINDOW_SIZE),
    C: make([]float64, constant.WINDOW_SIZE),
    strategies: []strategyFunc{
      // Add strategies here
      (*Asset).rand,
    },
  }
  a.close = a.closeFunc
  a.open = a.openFunc
  a.startStrategies()
  return
}

func (a *Asset) startStrategies() {
  n := len(a.strategies)
  a.channels = make([]chan struct{}, n)
  for i := range n {
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
    log.Panicln("SHUTTING DOWN")
  }
  pos := a.Positions[strat_name]
  pos.Rwm.Lock()
  defer pos.Rwm.Unlock()
  pos.Symbol = a.Symbol
  pos.AssetClass = a.Class
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

func (a *Asset) sendOpenOrder(order_type string, position_id string, symbol string, asset_class string, last_close float64) (string, int, error) {
  switch order_type {
    case "IOC":
      body, status, err := request.OpenLongIOC(symbol, asset_class, position_id, last_close)
      return body, status, err
  }
  return "", 0, nil
}

func (a *Asset) sendCloseOrder(open_side, order_type string, order_id string, symbol string, qty decimal.Decimal) (string, int, error) {
  var side string
  switch open_side {
  case "long":
    side = "sell"
  case "short":
    side = "buy"
  }
  switch order_type {
    case "IOC":
      body, status, err := request.CloseIOC(side, symbol, order_id, qty)
      return body, status, err
  }
  return "", 0, nil
}

//////////////////////// Methods below this point are for being called from strategy functions

func (a *Asset) i(num int) (index int) {
	index = constant.WINDOW_SIZE - 1 - num
	return
}

func (a *Asset) s(arr *[]float64, from int, to int) (slice []float64) {
  slice = (*arr)[(constant.WINDOW_SIZE - 1 - to):(constant.WINDOW_SIZE - from)]
  return
}

func (a *Asset) openChecks(strat_name string, trigger_time time.Time) bool {
  if NNP.Flag {
    return false
  }

  if _, ok := a.Positions[strat_name]; ok {
    return false
  }

  if a.ReceivedTime.Sub(a.Time) > constant.MAX_RECEIVED_TIME_DIFF_MS {
    log.Printf("[ CANCEL ]\t%s\t%s\tReceived time diff",
      util.AddWhitespace(a.Symbol, 10), strat_name,
    )
    return false
  } 

  if trigger_time.Sub(a.Time) > constant.MAX_TRIGGER_TIME_DIFF_MS {
    log.Printf("[ CANCEL ]\t%s\t%s\tTrigger time diff",
      util.AddWhitespace(a.Symbol, 10), strat_name,
    )
    return false
  }

  return true
}

func (a *Asset) sendOpen(order_type string, position_id string, symbol string, asset_class string, strat_name string, last_close float64) {
  // TODO: Log retries
  backoff_sec := 1.0
  retries := 0

  for {
    if retries > 1 {
      log.Println("[ CANCEL ]\t" +  util.AddWhitespace(symbol, 10) + "\tOpen failed on retry")
      a.removePosition(strat_name)
      return
    }

    body, status, err := a.sendOpenOrder(order_type, position_id, symbol, asset_class, last_close)
    if err != nil {
      util.Error(err, "Symbol", symbol, "Body", body)
      a.removePosition(strat_name)
      return
    }

    switch status {
    case 200:
      if retries >= 1 {
        log.Printf("[ INFO ]\t%s\t%s\tSending Open order successful on retry\n",
          util.AddWhitespace(symbol, 10), strat_name,
        )
      }
      return
    case 403:
      log.Printf("[ INFO ]\t%s\t%s\t%s\tForbidden block when sending Open order\tRetrying in (%.0f) seconds ...",
        util.AddWhitespace(symbol, 10), strat_name, body, backoff_sec,
      )
      util.Backoff(&backoff_sec)
    case 429:
      NNP.RateLimitSleep()
      util.Warning(errors.New("Rate limit exceeded on Open"),
        "Symbol", symbol, "Strat", strat_name,
        "Setting NoNewPositionsFlag to true for (seconds)", constant.RATE_LIMIT_SLEEP_SEC,
        util.AddWhitespace(symbol, 10), strat_name,
      )
    default:
      util.Error(errors.New("Sending open order failed"),
       "Symbol", symbol, "Status", status, "Retrying in (seconds)", backoff_sec,
      )
      util.Backoff(&backoff_sec)
    }

    retries++
  }
}

func (a *Asset) openFunc(side string, order_type string, strat_name string) {
  trigger_time := time.Now().UTC()
  if !a.openChecks(strat_name, trigger_time) {
    return
  }
  last_close := a.C[constant.WINDOW_SIZE-1]
  symbol := a.Symbol
  asset_class := a.Class
  position_id := a.createPositionID(strat_name)
  a.Mutex.Unlock()
  a.initiatePositionObject(strat_name, order_type, side, position_id, trigger_time)
  a.sendOpen(order_type, position_id, symbol, asset_class, strat_name, last_close)
  a.Mutex.Lock()
}

func (a *Asset) closeUpdatePosition(pos *Position, trigger_time time.Time, order_type string) (string, string, decimal.Decimal, string) {
  open_side := pos.OpenSide
  symbol := pos.Symbol
  qty := pos.Qty
  position_id := pos.PositionID
  pos.CloseOrderPending = true
  pos.CloseTriggerTime = trigger_time
  pos.CloseOrderType = order_type
  pos.CloseTriggerPrice = a.C[constant.WINDOW_SIZE-1]
  pos.ClosePriceTime = a.Time
  pos.ClosePriceReceivedTime = a.ReceivedTime
  return open_side, symbol, qty, position_id
}

func (a *Asset) sendClose(strat_name string, open_side string, order_type string, position_id string, symbol string, qty decimal.Decimal) {
  // TODD: Log retries
  backoff_sec := 1.0
  backoff_max := 20.0
  retries := 0
  
  for {
    if retries > 1 {
      NNP.NoNewPositionsTrue("Close")
      util.Info("Retry count", "Retries", retries)
    }

    body, status, err := a.sendCloseOrder(open_side, order_type, position_id, symbol, qty)
    if err != nil {
      util.Error(err, "Symbol", symbol, "Strat", strat_name)
    }

    switch status {
    case 200:
      if retries == 1 {
        log.Printf("[ INFO ]\t%s\t%s\tSending Close order succesfull on retry",
          util.AddWhitespace(symbol, 10), strat_name,
        )
      } else if retries > 1 {
        util.Info("Close successful after retries",
          "Symbol", symbol, "Strat", strat_name, "Retries", retries,
        )
        NNP.NoNewPositionsFalse("Close")
      }
      return
    case 403:
      log.Printf("[ INFO ]\t%s\t%s\t%s\tForbidden block on Close\tRetrying in (%.0f) seconds ...",
        util.AddWhitespace(symbol, 10), strat_name, body, backoff_sec,
      )
      util.BackoffWithMax(&backoff_sec, backoff_max)
    case 422:
      util.Error(errors.New("Close order unprocessable"),
        "Symbol", symbol, "Strat", strat_name, "Body", body, "Retrying in (seconds)", backoff_sec,
      )
      return
    case 429:
      NNP.RateLimitSleep()
      util.Warning(errors.New("Rate limit exceeded on Close"),
        "Symbol", symbol, "Strat", strat_name, "Retrying in (seconds)", backoff_sec,
        "Setting NoNewPositionsFlag to true for (seconds)", constant.RATE_LIMIT_SLEEP_SEC,
        util.AddWhitespace(symbol, 10), strat_name, backoff_sec,
      )
      util.BackoffWithMax(&backoff_sec, backoff_max)
    default:
      util.Error(errors.New("Sending close order failed"),
        "Symbol", symbol, "Status", status, "Retrying in (seconds)", backoff_sec,
      )
      util.BackoffWithMax(&backoff_sec, backoff_max)
    }

    retries++
  }
}

func (a *Asset) closeFunc(order_type string, strat_name string) {
  trigger_time := time.Now().UTC()
  if _, ok := a.Positions[strat_name]; !ok {
    return
  }
  pos := a.Positions[strat_name]
  pos.Rwm.Lock()
  if pos.CloseOrderPending || pos.OpenOrderPending {
    log.Printf("[ INFO ]\t%s\t%s\tClose cancelled due to order pending", util.AddWhitespace(a.Symbol, 10), strat_name)
    pos.Rwm.Unlock()
    return
  }
  open_side, symbol, qty, position_id := a.closeUpdatePosition(pos, trigger_time, order_type)
  pos.Rwm.Unlock()
  a.sendClose(strat_name, open_side, order_type, position_id, symbol, qty)
}

func (a *Asset) priceDeviation(fill_price float64) float64 {
  if fill_price == 0 {
    return 0
  }
  return (a.C[a.i(0)] / fill_price - 1) * 100
}

func (a *Asset) stopLoss(percent float64, strat_name string) {
  // TODO:
  //   -> Log if stop loss triggered to db
  //   -> Implement logic for short positions (This requires significant changes throughout the code)
  if _, ok := a.Positions[strat_name]; !ok {
    return
  }
  if a.priceDeviation(a.Positions[strat_name].OpenFilledAvgPrice) < (percent * -1) {
    a.close("IOC", strat_name)
    log.Printf("[ INFO ]\t%s\t%s\tStopLoss", a.Symbol, strat_name)
  }
}

func (a *Asset) takeProfit(percent float64, strat_name string) {
  // TODO:
  //   -> Log if take profit triggered to db
  //   -> Implement logic for short positions
  if _, ok := a.Positions[strat_name]; !ok {
    return
  }
  if a.priceDeviation(a.Positions[strat_name].OpenFilledAvgPrice) > percent {
    a.close("IOC", strat_name)
    log.Printf("[ INFO ]\t%s\t%s\tTakeProfit", a.Symbol, strat_name)
  }
}

func (a *Asset) trailingStop(percent float64, strat_name string) {
  if _, ok := a.Positions[strat_name]; !ok {
    return
  }
  pos := a.Positions[strat_name]

  p := a.C[a.i(0)]
  if p > pos.TrailingStopBase {
    pos.TrailingStopBase = p
    return
  }

  if a.priceDeviation(pos.TrailingStopBase) < (percent * -1) {
    a.close("IOC", strat_name)
    log.Printf("[ INFO ]\t%s\t%s\tTrailingStop", a.Symbol, strat_name)
  }
}
