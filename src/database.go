package main

import (
  "fmt"
  "sync"
  "log"
  "time"
  "errors"
  "database/sql"
  _ "github.com/go-sql-driver/mysql"
  "github.com/shopspring/decimal"
  "github.com/Kjellemann1/AlgoTrader-Go/util"
  "github.com/Kjellemann1/AlgoTrader-Go/constant"
  "github.com/Kjellemann1/AlgoTrader-Go/request"
)

type Query struct {
  Action            string
  PositionID        string  // "client_order_id"
  Symbol            string
  AssetClass        string
  Side              string  // "buy", "sell"
  StratName         string
  OrderType         string
  Qty               decimal.Decimal
  PriceTime         time.Time
  ReceivedTime      time.Time
  TriggerTime       time.Time
  TriggerPrice      float64
  FillTime          time.Time
  FilledAvgPrice    float64
  TrailingStop      float64
  BadForAnalysis    bool
  TrailingStopPrice float64
  NCloseOrders      int8
}

type Database struct {
  conn                  *sql.DB
  db_chan               chan *Query
  insert_trade          *sql.Stmt
  insert_position       *sql.Stmt
  delete_position       *sql.Stmt
  update_trailing_stop  *sql.Stmt
  update_n_close_orders *sql.Stmt
}

func NewDatabase(db_chan chan *Query) (db *Database) {
  db = &Database{}
  db.db_chan = db_chan
  return
}

func (db *Database) prepQueries() error {
  var err error
  db.insert_trade, err = db.conn.Prepare(`
    INSERT INTO trades (
      action,
      position_id,
      symbol,
      asset_class,
      side, 
      strat_name,
      order_type,
      qty,
      price_time,
      received_time,
      trigger_time,
      trigger_price,
      fill_time,
      filled_avg_price,
      bad_for_analysis,
      n_close_orders
    ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);
  `)
  if err != nil {
    return err
  }
  db.insert_position, err = db.conn.Prepare(`
    INSERT INTO positions (
      position_id,
      symbol,
      asset_class,
      side,
      strat_name,
      order_type,
      qty,
      price_time,
      received_time,
      trigger_time,
      trigger_price,
      fill_time,
      filled_avg_price,
      trailing_stop,
      bad_for_analysis,
      n_close_orders
    ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);
  `)
  if err != nil {
    return err
  }
  db.delete_position, err = db.conn.Prepare(`
    DELETE FROM positions WHERE symbol = ? AND strat_name = ?;
  `)
  if err != nil {
    return err
  }
  db.update_trailing_stop, err = db.conn.Prepare(`
    UPDATE positions SET trailing_stop = ? WHERE symbol = ? AND strat_name = ?;
  `)
  if err != nil {
    return err
  }
  db.update_n_close_orders, err = db.conn.Prepare(`
    UPDATE positions SET n_close_orders = ? WHERE symbol = ? AND strat_name = ?;
  `)
  if err != nil {
    return err
  }
  return nil
}

func (db *Database) pingAndSetupQueries() error {
  // Ping() automatically tries to establish a connection if necessary
  err := db.conn.Ping()
  if err != nil {
    return err
  }
  err = db.prepQueries()
  if err != nil {
    return err
  }
  return nil
}

func (db *Database) errorHandler(
  err error, func_name string, response sql.Result, query *Query, retries int, backoff_sec *int,
) {
  util.Warning2(err, "Query", *query, "Response", response, "Retries", retries)
  err = db.pingAndSetupQueries()
  if err != nil {
    NNP.NoNewPositionsTrue("Database")
    if retries <= 3 {
      util.Warning(err, "Setting NO_NEW_TRADES == true", "Retrying in (seconds)", *backoff_sec)
      util.Backoff(backoff_sec)
      db.errorHandler(err, func_name, response, query, retries + 1, backoff_sec)
    } else {
      util.Error(err, "MAX RETRIES REACHED", retries, "CLOSING ALL POSITIONS AND SHUTTING DOWN", "...")
      request.CloseAllPositions(2, 0)
      log.Panicln("SHUTTING DOWN")
    }
  }
  if retries > 3 {
    util.Warning(nil, "MAX RETRIES REACHED", retries, "CLOSING ALL POSITIONS AND SHUTTING DOWN", "...")
    request.CloseAllPositions(2, 0)
    log.Panicln("SHUTTING DOWN")
  }
  db.queryHandler(query, *backoff_sec, retries + 1)
  // Code below is not reached if query fails
  if retries > 0 {
    util.Info("Query successful after retries", "Retries", retries + 1)
  }
  util.Info("Query successful")
  NNP.NoNewPositionsFalse("Database")
}

func (db *Database) insertTrade(query *Query, backoff_sec int, retries int) {
  response, err := db.insert_trade.Exec(
    query.Action,
    query.PositionID,
    query.Symbol,
    query.AssetClass,
    query.Side,
    query.StratName,
    query.OrderType,
    query.Qty, 
    query.PriceTime,
    query.ReceivedTime,
    query.TriggerTime,
    query.TriggerPrice,
    query.FillTime,
    query.FilledAvgPrice,
    query.BadForAnalysis,
    query.NCloseOrders,
  )
  if err != nil {
    db.errorHandler(err, "insertTrade", response, query, retries, &backoff_sec)
  }
}

func (db *Database) insertPosition(query *Query, backoff_sec int, retries int) {
  response, err := db.insert_position.Exec(
    query.PositionID,
    query.Symbol,
    query.AssetClass,
    query.Side,
    query.StratName,
    query.OrderType,
    query.Qty,
    query.PriceTime,
    query.ReceivedTime,
    query.TriggerTime,
    query.TriggerPrice,
    query.FillTime,
    query.FilledAvgPrice,
    query.TrailingStop,
    query.BadForAnalysis,
    query.NCloseOrders,
  )
  if err != nil {
    db.errorHandler(err, "insertPosition", response, query, retries, &backoff_sec)
  }
}

func (db *Database) deleteAllPositions(backoff_sec int, retries int) {
  response, err := db.conn.Exec("DELETE FROM positions WHERE 1;")
  if err != nil {
    db.errorHandler(err, "deleteAllPositions", response, nil, retries, &backoff_sec)
  } else {
    log.Println("[ OK ]\tAll positions deleted from database")
  }
}

func (db *Database) deletePosition(query *Query, backoff_sec int, retries int) {
  response, err := db.delete_position.Exec(query.Symbol, query.StratName)
  if err != nil {
    db.errorHandler(err, "deletePosition", response, query, retries, &backoff_sec)
  }
}

func (db *Database) updateTrailingStop(query *Query, backoff_sec int, retries int) {
  response, err := db.update_trailing_stop.Exec(query.TrailingStopPrice, query.Symbol, query.StratName)
  if err != nil {
    db.errorHandler(err, "updateTrailingStop", response, query, retries, &backoff_sec)
  }
}

func (db *Database) updateNCloseOrders(query *Query, backoff_sec int, retries int) {
  response, err := db.update_n_close_orders.Exec(query.NCloseOrders, query.Symbol, query.StratName)
  if err != nil {
    db.errorHandler(err, "updateNCloseOrders", response, query, retries, &backoff_sec)
  }
}

func (db *Database) retrievePositions() {
  response, err := db.conn.Query("SELECT * FROM positions;")
  if err != nil {
    util.Error(err, "retrievePositions")
  }
  defer response.Close()
  for response.Next() {
    var positionID, symbol, assetClass, side, stratName, orderType string
    var qty, triggerPrice, filledAvgPrice, trailingStop float64
    var priceTime, receivedTime, triggerTime, fillTime time.Time
    var badForAnalysis bool
    var nCloseOrders int8
    err = response.Scan(
      &positionID,
      &symbol,
      &assetClass,
      &side,
      &stratName,
      &orderType,
      &qty,
      &priceTime,
      &receivedTime,
      &triggerTime,
      &triggerPrice,
      &fillTime,
      &filledAvgPrice,
      &trailingStop,
      &badForAnalysis,
      &nCloseOrders,
    )
    if err != nil {
      util.Error(err, "retrievePositions")
    }
    fmt.Println(positionID, symbol, assetClass, side, stratName, orderType, qty, priceTime, receivedTime, triggerTime, triggerPrice, fillTime, filledAvgPrice, trailingStop, badForAnalysis, nCloseOrders)
  }
}

func (db *Database) queryHandler(query *Query, backoff_sec int, retries int) {
  switch query.Action {
    case "open":
      db.insertTrade(query, backoff_sec, retries)
      db.insertPosition(query, backoff_sec, retries)
    case "close":
      db.insertTrade(query, backoff_sec, retries)
      if !query.Qty.IsZero() {
        db.updateNCloseOrders(query, backoff_sec, retries)
      } else {
        db.deletePosition(query, backoff_sec, retries)
      }
    case "update":
      db.updateTrailingStop(query, backoff_sec, retries)
    case "delete_all_positions":
      db.deleteAllPositions(backoff_sec, retries)
    case "retrieve_positions":
      db.retrievePositions()
    default:
      util.Error(errors.New("Invalid query type"), "Query", query)
  }
}

func (db *Database) listen() {
  log.Println("[ OK ]\tDatabase listening")
  for {
    query := <-db.db_chan
    if query == nil {
      db.conn.Close()
      return
    }
    db.queryHandler(query, 5, 0)
  }
}

func (db *Database) connect() {
  url := fmt.Sprintf("%s:%s@/%s", constant.DB_USER, constant.DB_PASSWORD, constant.DB_NAME)
  var err error
  db.conn , err = sql.Open("mysql", url)
  if err != nil {
    panic(err.Error())
  }
}

func (db *Database) Start(wg *sync.WaitGroup, assets map[string]map[string]*Asset) {
  defer wg.Done()
  db.connect()
  defer db.conn.Close()
  db.RetrieveState(assets)
  err := db.prepQueries()
  if err != nil {
    log.Panicf("[ ERROR ]\tsetupQueries() failed: %s\n", err.Error())
  }
  db.listen()
}

func (db *Database) RetrieveState(assets map[string]map[string]*Asset) {
  globRwm.Lock()
  defer globRwm.Unlock()
  response, err := db.conn.Query("SELECT * FROM positions;")
  if err != nil {
    util.ErrorPanic(err)
  }
  defer response.Close()
  for response.Next() {
    var positionID, symbol, assetClass, side, stratName, orderType string
    var qty decimal.Decimal
    var triggerPrice, filledAvgPrice, trailingStop float64
    var priceTime, receivedTime, triggerTime, fillTime time.Time
    var badForAnalysis bool
    var nCloseOrders int8
    err = response.Scan(
      &symbol,
      &stratName,
      &assetClass,
      &positionID,
      &side,
      &orderType,
      &qty,
      &priceTime,
      &triggerTime,
      &triggerPrice,
      &fillTime,
      &filledAvgPrice,
      &trailingStop,
      &badForAnalysis,
      &receivedTime,
      &nCloseOrders,
    )
    assets[assetClass][symbol].Positions[stratName] = &Position{
      Symbol: symbol,
      AssetClass: assetClass,
      StratName: stratName,
      Qty: qty,
      BadForAnalysis: badForAnalysis,
      PositionID: positionID,
      OpenOrderPending: false,
      OpenTriggerTime: triggerTime,
      OpenSide: side,
      OpenOrderType: orderType,
      OpenTriggerPrice: triggerPrice,
      OpenPriceTime: priceTime,
      OpenPriceReceivedTime: receivedTime,
      OpenFillTime: fillTime,
      OpenFilledAvgPrice: filledAvgPrice,
      CloseOrderPending: false,
      NCloseOrders: nCloseOrders,
    }
    assets[assetClass][symbol].AssetQty = assets[assetClass][symbol].AssetQty.Add(qty)
    log.Printf("[ INFO ]\t Retrieved position: %s %s %s", symbol, stratName, qty.String())
  }
  log.Println("[ OK ]\tState retrieved from database")
}
