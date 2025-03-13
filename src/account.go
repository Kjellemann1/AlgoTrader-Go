package main

import (
  "fmt"
  "log"
  "regexp"
  "errors"
  "strconv"
  "sync"
  "time"
  "context"
  "net/http"
  "github.com/gorilla/websocket"
  "github.com/valyala/fastjson"
  "github.com/shopspring/decimal"
  "github.com/Kjellemann1/AlgoTrader-Go/constant"
  "github.com/Kjellemann1/AlgoTrader-Go/request"
  "github.com/Kjellemann1/AlgoTrader-Go/util"
)

type Account struct {
  conn    *websocket.Conn
  parser  fastjson.Parser
  db_chan chan *Query
  assets  map[string]map[string]*Asset
  url     string

  listen func(*sync.WaitGroup, chan int8)
  pingPong func(*sync.WaitGroup, context.Context, chan int8)
}

func NewAccount(assets map[string]map[string]*Asset, url string, db_chan chan *Query) *Account {
  a := &Account{
    parser: fastjson.Parser{},
    assets: assets,
    db_chan: db_chan,
    url: url,
  }
  a.pingPong = a.pingPongFunc
  a.listen = a.listenFunc
  return a
}

func (a *Account) onAuth(message []byte) (string, error) {
  parsed_msg, err := a.parser.ParseBytes(message)
  if err != nil {
    util.Error(err)
    return "", err
  }

  data := parsed_msg.Get("data")
  if string(data.GetStringBytes("status")) == "authorized" {
    util.Ok("Authenticated with account websocket")
    return "", nil
  } else {
    action := string(data.GetStringBytes("action"))
    if action == "listen" {
      return action, nil
    } else {
      return "", errors.New("Failed to authenticate with account websocket")
    }
  }
}

func (a *Account) messageHandler(message []byte) error {
  parsed_msg, err := a.parser.ParseBytes(message)
  if err != nil {
    util.Error(err)
    return err
  }

  message_type := string(parsed_msg.GetStringBytes("stream"))
  switch message_type {
    case "trade_updates":
      update := a.updateParser(parsed_msg)
      if update == nil {
        return nil
      }
      a.orderUpdateHandler(update)
    case "listening":
      util.Ok("Listening to order updates")
  }

  return nil
}

func (a *Account) connect() (err error) {
  var response *http.Response
  a.conn, response, err = websocket.DefaultDialer.Dial(a.url, nil)
  if err != nil {
    log.Println("Response", response.Body)
    return err
  }

  err = a.conn.WriteMessage(websocket.TextMessage, 
    fmt.Appendf(make([]byte, 0), `{"action":"auth","key":"%s","secret":"%s"}`,
    constant.KEY, constant.SECRET),
  )
  if err != nil {
    return err
  }

  for {
    _, message, err := a.conn.ReadMessage()
    if err != nil {
      log.Println("Message", string(message))
      return err
    }

    action, err := a.onAuth(message)
    if err != nil{
      return err
    } else if action == "listen" {
      continue
    }

    break
  }

  err = a.conn.WriteMessage(websocket.TextMessage,
    []byte(`{"action":"listen","data":{"streams":["trade_updates"]}}`),
  )
  if err != nil {
    return err
  }

  return nil
}

func (a *Account) pingPongFunc(connWg *sync.WaitGroup, ctx context.Context, err_chan chan int8) {
  defer connWg.Done()

  if err := a.conn.SetReadDeadline(time.Now().Add(constant.READ_DEADLINE_SEC)); err != nil {
    util.Warning(err)
  }

  a.conn.SetPongHandler(func(string) error {
    err := a.conn.SetReadDeadline(time.Now().Add(constant.READ_DEADLINE_SEC))
    if err != nil {
      util.Warning(err)
    }
    return nil
  })

  ticker := time.NewTicker(constant.PING_INTERVAL_SEC)
  defer ticker.Stop()
  util.Ok("PingPong initiated for account websocket")

  for {
    select {
    case <-ctx.Done():
      return
    case <-ticker.C:
      if err := a.conn.WriteControl(
        websocket.PingMessage, []byte("ping"), 
        time.Now().Add(5 * time.Second)); err != nil {
        select {
        case err_chan <-1:
          util.Error(err)
        default:
        }
        return
      }
    }
  }
}

func (a *Account) listenFunc(connWg *sync.WaitGroup, err_chan chan int8) {
  defer connWg.Done()

  for {
    _, message, err := a.conn.ReadMessage()
    if err != nil {
      select {
      case err_chan <-1:
        util.Error(err)
      default:
      }
      return
    }

    _ = a.messageHandler(message)
  }
}

func (a *Account) start(wg *sync.WaitGroup, ctx context.Context, backoff_sec_min float64) {
  defer wg.Done()

  backoff_sec := backoff_sec_min
  retries := 0

  for {
    if err := a.connect() ; err != nil {
      if retries < 5 {
        util.Error(err, "Retries", retries)
        util.Backoff(&backoff_sec)
        retries++
        continue
      } else {
        NNP.NoNewPositionsTrue("")
        util.Error(err, "Max retries reached", retries, "CLOSING ALL POSITIONS AND SHUTTING DOWN", "...")
        request.CloseAllPositions(2, 0)
        log.Panicln("SHUTTING DOWN")
      }
    }

    backoff_sec = backoff_sec_min
    retries = 0

    err_chan := make(chan int8)

    a.checkPending()

    var connWg sync.WaitGroup

    connWg.Add(1)
    go a.listen(&connWg, err_chan)

    context, cancel := context.WithCancel(ctx)
    connWg.Add(1)
    go a.pingPong(&connWg, context, err_chan)

    select {
    case <-ctx.Done():
      cancel()
      a.conn.Close()
      connWg.Wait()
      return
    case <-err_chan:
      cancel()
      a.conn.Close()
      connWg.Wait()
    }
  }
}

type OrderUpdate struct {
  Event          *string
  AssetClass     *string
  StratName      *string
  Side           *string
  Symbol         *string
  AssetQty       *decimal.Decimal
  FillTime       *time.Time
  FilledAvgPrice *float64
}

func (a *Account) getEvent(data *fastjson.Value) *string {
  // Shutdown if nil
  // Only handle fill, partial_fill and canceled events.
  // Other events are likely not relevant. https://alpaca.markets/docs/api-documentation/api-v2/streaming/
  event := data.GetStringBytes("event")
  if event == nil {
    NNP.NoNewPositionsTrue("")
    util.Error(
      errors.New("EVENT NOT IN TRADE UPDATE"), "Parsed message", data.String(),
      "CLOSING ALL POSITIONS AND SHUTTING DOWN", "...",
    )
    request.CloseAllPositions(2, 0)
    log.Panicln("SHUTTING DOWN")
  }

  event_str := string(event)
  if event_str != "fill" && event_str != "partial_fill" && event_str != "canceled" {
    return nil
  }

  return &event_str
}

func (p *Account) getPositionID(order *fastjson.Value) *string {
  position_id := order.GetStringBytes("client_order_id")  // client_order_id == PositionID
  if position_id == nil {
    util.Warning(errors.New("PositionID not found in order update"), nil)
    return nil
  }

  position_id_str := string(position_id)

  return &position_id_str
}

func grepStratName(position_id *string) *string {
  pattern := `strat\[(.*?)\]`
  re := regexp.MustCompile(pattern)
  match := re.FindStringSubmatch(*position_id)
  if len(match) > 1 {
    return &match[1]
  }

  return nil
}

func (a *Account) getStratName(order *fastjson.Value) *string {
  // Return if nil 
  position_id := a.getPositionID(order)

  if position_id == nil {
    return nil
  }

  return grepStratName(position_id)
}

func (a *Account) getAssetClass(order *fastjson.Value) *string {
  // Shutdown if nil
  asset_class := order.GetStringBytes("asset_class")
  if asset_class == nil {
    NNP.NoNewPositionsTrue("")
    util.Error(
      errors.New("ASSET CLASS NOT IN TRADE UPDATE"), "Parsed message", order.String(),
      "CLOSING ALL POSITIONS AND SHUTTING DOWN", "...",
    )
    request.CloseAllPositions(2, 0)
    log.Panicln("SHUTTING DOWN")
  }

  asset_class_str := string(asset_class)
  if asset_class_str == "us_equity" {
    asset_class_str = "stock"
  }

  return &asset_class_str
}

func (a *Account) getSymbol(order *fastjson.Value) *string {
  // Return if nil, shutdown if fails
  symbol := order.GetStringBytes("symbol")
  if symbol == nil {
    NNP.NoNewPositionsTrue("")
    util.Error(
      errors.New("SYMBOL NOT IN TRADE UPDATE"), "Parsed message", order.String(),
      "CLOSING ALL POSITIONS AND SHUTTING DOWN", "...",
    )
    request.CloseAllPositions(2, 0)
    log.Panicln("SHUTTING DOWN")
  }

  symbol_str := string(symbol)

  return &symbol_str
}

func (a *Account) getSide(order *fastjson.Value) *string {
  side := order.GetStringBytes("side")
  if side == nil {
    return nil
  }

  side_str := string(side)

  return &side_str
}

func (a *Account) getAssetQty(data *fastjson.Value) *decimal.Decimal {
  // Return if nil, shutdown if fails
  asset_qty := data.GetStringBytes("position_qty")
  if asset_qty == nil {
    return nil
  }

  asset_qty_dec, err := decimal.NewFromString(string(asset_qty))
  if err != nil {
    NNP.NoNewPositionsTrue("")
    util.Error(err, "Asset qty", asset_qty, "CLOSING ALL POSITIONS AND SHUTTING DOWN", "...")
    request.CloseAllPositions(2, 0)
    log.Panicln("SHUTTING DOWN")
  }

  return &asset_qty_dec
}

func (a *Account) getFillTime(order *fastjson.Value) *time.Time {
  fill_time := order.GetStringBytes("filled_at")
  if fill_time == nil {
    return nil
  }

  fill_time_t, err := time.Parse(time.RFC3339, string(fill_time))
  if err != nil {
    util.Warning(errors.New("Failed to convert fill_time to time.Time"))
  }

  return &fill_time_t
}

func (a *Account) getFilledAvgPrice(order *fastjson.Value) *float64 {
  filled_avg_price := order.GetStringBytes("filled_avg_price")
  if filled_avg_price == nil {
    return nil
  }

  filled_avg_price_float, err := strconv.ParseFloat(string(filled_avg_price), 64)
  if err != nil {
    util.Warning(errors.New("Failed to convert filled_avg_price to float"))
  }

  return &filled_avg_price_float
}

func (a *Account) updateParser(pm *fastjson.Value) *OrderUpdate {
  data := pm.Get("data")
  order := data.Get("order")

  event := a.getEvent(data)
  if event == nil {
    return nil
  }

  strat_name := a.getStratName(order)
  if strat_name == nil {
    return nil
  }

  asset_qty := a.getAssetQty(data)
  asset_class := a.getAssetClass(order)
  symbol := a.getSymbol(order)
  side := a.getSide(order)
  fill_time := a.getFillTime(order)
  filled_avg_price := a.getFilledAvgPrice(order)

  return &OrderUpdate {
    Event:            event,
    AssetClass:       asset_class,
    StratName:        strat_name,
    Side:             side,
    Symbol:           symbol,
    AssetQty:         asset_qty,
    FillTime:         fill_time,
    FilledAvgPrice:   filled_avg_price,
  }
}

func updateAssetQty(p *Position, a *Asset, u *OrderUpdate) {
  a.Rwm.Lock()
  defer a.Rwm.Unlock()

  var position_change decimal.Decimal = (*u.AssetQty).Sub(a.Qty)
  p.Qty = p.Qty.Add(position_change)
  a.Qty = *u.AssetQty

  if !a.sumPosQtysEqAssetQty() {
    util.Error(
      errors.New("Sum of position qty not equal to asset qty"),
      "Asset", a.Qty, "Position", p.Qty, "OrderUpdate", u,
      "CLOSING ALL POSITIONS AND SHUTTING DOWN", "...",
    )
    request.CloseAllPositions(2, 0)
    log.Fatal("SHUTTING DOWN")
  }
}

func (a *Account) closeLogic(asset *Asset, pos *Position, u *OrderUpdate) {
  if u.FilledAvgPrice != nil {
    pos.CloseFilledAvgPrice = *u.FilledAvgPrice
  }

  if u.FillTime != nil {
    pos.CloseFillTime = *u.FillTime
  }

  if *u.Event == "fill" || *u.Event == "canceled" {
    a.db_chan <-pos.LogClose()
    if pos.Qty.IsZero() {
      asset.removePosition(*u.StratName)
    } else {
      asset.Mutex.Lock()
      pos.CloseOrderPending = false
      asset.close("IOC", *u.StratName)
      asset.Mutex.Unlock()
    }
  }
}

func (a *Account) openLogic(asset *Asset, pos *Position, u *OrderUpdate) {
  if u.FilledAvgPrice != nil {
    pos.OpenFilledAvgPrice = *u.FilledAvgPrice
  }

  if u.FillTime != nil {
    pos.OpenFillTime = *u.FillTime
  }

  if *u.Event == "fill" || *u.Event == "canceled" {
    if pos.Qty.IsZero() {
      asset.removePosition(*u.StratName)
    } else {
      a.db_chan <-pos.LogOpen()
      pos.OpenOrderPending = false
    }
  }
}

func (a *Account) reconnectDiff(u *OrderUpdate) {
  asset := a.assets[*u.AssetClass][*u.Symbol]
  asset.Rwm.Lock()
  defer asset.Rwm.Unlock()

  if !asset.sumPosQtysEqAssetQty() {
    NNP.NoNewPositionsTrue("")
    util.Error(errors.New("Position quantities do not sum to asset qty"),
      "CLOSING ALL POSITIONS AND SHUTTING DOWN", "...",
    )
    request.CloseAllPositions(2, 0)
    log.Panicln("SHUTTING DOWN")
  }
}

func (a *Account) orderUpdateHandler(u *OrderUpdate) {
  if *u.StratName == "reconnect_multiple_diff" {
    a.reconnectDiff(u)
    return
  }

  var asset = a.assets[*u.AssetClass][*u.Symbol]
  var pos *Position = asset.Positions[*u.StratName]

  if pos == nil {
    util.Error(errors.New("Position nil: %s"),
      "Symbol", *u.Symbol,
      "StratName", *u.StratName,
      "CLOSING ALL POSITIONS AND SHUTTING DOWN", "...",
    )
    request.CloseAllPositions(2, 0)
    log.Panicln("SHUTTING DOWN")
  }

  pos.Rwm.Lock()
  defer pos.Rwm.Unlock()

  if u.AssetQty != nil {
    updateAssetQty(pos, asset, u)
  }

  if pos.OpenOrderPending {
    a.openLogic(asset, pos, u)
  } else if pos.CloseOrderPending {
    a.closeLogic(asset, pos, u)
  }
}

// When a reconnect is triggered in account, there could be missed order updates.
// Therefore we need to check for any orders that where executed while the connection
// was down, and update the positions accordingly.

type ParsedClosedOrder struct {
  Side *string
  StratName *string
  Symbol *string
  FilledAvgPrice *float64
  FilledQty *decimal.Decimal
  FillTime *time.Time
}

func getString(co *fastjson.Value, element string) *string {
  byte := co.GetStringBytes(element)
  if byte == nil {
    return nil
  }
  str := string(byte)
  return &str
}

func getStratName(co *fastjson.Value) *string {
  byte := co.GetStringBytes("client_order_id")
  if byte == nil {
    return nil
  }
  str := string(byte)
  return grepStratName(&str)
}

func getFloat(co *fastjson.Value, element string) *float64 {
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

func getFilledQty(co *fastjson.Value) *decimal.Decimal {
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

func getFillTime(co *fastjson.Value) *time.Time {
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

func parse(co *fastjson.Value) *ParsedClosedOrder {
  return &ParsedClosedOrder{
    Symbol: getString(co, "symbol"),
    Side: getString(co, "side"),
    StratName: getStratName(co),
    FilledAvgPrice: getFloat(co, "filled_avg_price"),
    FilledQty: getFilledQty(co),
    FillTime: getFillTime(co),
  }
}

func (a *Account) parseClosedOrders(relevant map[string][]*fastjson.Value) map[string][]*ParsedClosedOrder {
  parsed := make(map[string][]*ParsedClosedOrder)
  for symbol, arr := range relevant {
    for _, co := range arr {
      parsed[symbol] = append(parsed[symbol], parse(co))
    }
  }

  return parsed
}

func (a *Account) sendCloseGTC(diff decimal.Decimal, symbol string, backoff_sec float64) {
  retries := 0
  for {
    status, err := request.CloseGTC("sell", symbol, "strat[reconnect_multiple_diff]", diff)
    if err != nil {
      util.Error(err, "Failed to send close order", "...")
    }
    switch status {
    case 403:
      util.Warning(errors.New("forbidden block"),
        "Retries", retries,
        "Trying again in (seconds)", backoff_sec,
      )
      util.Backoff(&backoff_sec)
    case 200:
      if retries > 0 {
        util.Warning(errors.New("order sent after retries"), "Retries", retries)
      }
      util.Ok("Reconciliation close order sent")
      return
    default:
      util.Error(fmt.Errorf("failed to send close order. Status: %d", status),
        "Retries", retries,
        "Trying again in (seconds)", backoff_sec,
      )
      util.Backoff(&backoff_sec)
    }
    if retries >= constant.REQUEST_RETRIES {
      util.Error(errors.New("max retries reached"),
        "CLOSING ALL POSITIONS AND SHUTTING DOWN", "...",
      )
      request.CloseAllPositions(2, 0)
      log.Panicln("SHUTTING DOWN")
    }
    retries++
  }
}

func (a *Account) multiple(parsed []*ParsedClosedOrder, asset_class string) {
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
    a.sendCloseGTC(diff_no_pending, *parsed[0].Symbol, 0)
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
  a.db_chan <-pos.LogOpen()
}

func (a *Account) diffNegative(diff decimal.Decimal, asset_class string, parsed []*ParsedClosedOrder) {
  pco := parsed[0]
  asset := a.assets[asset_class][*pco.Symbol]
  pos := asset.Positions[*pco.StratName]
  pos.BadForAnalysis = true
  if !diff.Abs().Equal(pos.Qty) {
    pos.Qty = pos.Qty.Add(diff)
    pos.CloseOrderPending = false
    a.db_chan <-pos.LogClose()
    asset.Mutex.Lock()
    asset.close("IOC", pos.StratName)
    asset.Mutex.Unlock()
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
  pos.CloseOrderPending = false
  a.db_chan <-pos.LogClose()
  asset.Mutex.Lock()
  asset.close("IOC", pos.StratName)
  asset.Mutex.Unlock()
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
    for symbol, pcos := range parsed {
      if len(pcos) > 1 {
        a.multiple(pcos, asset_class)
        continue
      }

      diff := qtys[symbol].Sub((*a.assets[asset_class][symbol]).Qty)
      a.assets[asset_class][symbol].Qty = qtys[symbol]

      switch {
      case diff.IsPositive():
        a.diffPositive(diff, asset_class, pcos)
      case diff.IsNegative():
        a.diffNegative(diff, asset_class, pcos)
      default:
        a.diffZero(asset_class, pcos)
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
        fill_time := m.GetStringBytes("filled_at")

        if position_id == nil {
          util.Warning(errors.New("PositionID not found"), nil)
          continue
        } else if symbol == nil {
          util.Warning(errors.New("symbol not found"), nil)
          continue
        } else if string(fill_time) == "null" {
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
    util.Ok("No pending orders")
    return
  }

  arr, err := request.GetClosedOrders(positionsSymbols(pending), 5, 0)
  if err != nil {
    NNP.NoNewPositionsTrue("")
    util.Error(err, "CLOSING ALL POSITIONS AND SHUTTING DOWN", "...")
    request.CloseAllPositions(2, 0)
    log.Panicln("SHUTTING DOWN")
  }

  relevant := a.filterRelevantOrders(arr, pending)
  if len(relevant) == 0 {
    util.Ok("No pending orders closed")
    return
  }

  parsed := a.parseClosedOrders(relevant)

  a.updatePositions(parsed)

  util.Ok("Pending orders updated")
}
