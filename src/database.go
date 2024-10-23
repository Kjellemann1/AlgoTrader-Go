
package src

import (
  "fmt"
  "sync"
  "log"
  "time"
  "database/sql"
  "github.com/shopspring/decimal"
  _ "github.com/go-sql-driver/mysql"
  "github.com/Kjellemann1/AlgoTrader-Go/src/util/backoff"
  "github.com/Kjellemann1/AlgoTrader-Go/src/constant"
  "github.com/Kjellemann1/AlgoTrader-Go/src/util/push"
  "github.com/Kjellemann1/AlgoTrader-Go/src/order"
)


type Query struct {
  Action            string
  PositionID        string
  Symbol            string
  AssetClass        string
  Side              string  // "buy", "sell"
  StratName         string
  OrderType         string
  Qty               decimal.Decimal  // Stored as a string to avoid floating point errors
  PriceTime         time.Time
  TriggerTime       time.Time
  TriggerPrice      float64
  FillTime          time.Time
  FilledAvgPrice    float64
  OrderSentTime     time.Time
  TrailingStop      float64
  BadForAnalysis    bool
  TrailingStopPrice float64
}


type Database struct {
  conn                 *sql.DB
  db_chan              chan *Query
  insert_trade         *sql.Stmt
  insert_position      *sql.Stmt
  delete_position      *sql.Stmt
  update_trailing_stop *sql.Stmt
}


// Prepare queries so they don't have to be created every for every query
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
      trigger_time,
      trigger_price,
      fill_time,
      filled_avg_price,
      order_sent_time,
      bad_for_analysis
    ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);
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
      trigger_time,
      trigger_price,
      fill_time,
      filled_avg_price,
      order_sent_time,
      trailing_stop,
      bad_for_analysis
    ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);
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
  return nil
}


// Reestablish connection if lost
func (db *Database) pingAndSetupQueries() error {
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
  // TODO: Implement specific error handling for bad queries
  err error, func_name string, response sql.Result, query *Query, retries int, backoff_sec *int,
) {
  if retries == 0 {
    push.Warning("Failed to execute query", err)
    log.Printf(
      "[ WARNING ]\t%s failed\n" +
      "  -> Retries: %d\n" +
      "  -> Query: %v\n" +
      "  -> Error: %s\n" +
      "  -> Response: %s\n",
      func_name, retries, *query, err.Error(), response,
    )
  } else {
    push.Error("Failed to execute query on retry", err)
    log.Printf(
      "[ ERROR ]\t%s failed\n" +
      "  -> Retries: %d\n" +
      "  -> Error: %s\n",
      func_name, retries, err.Error(),
    )
  }
  // Ping database. Ping() should automatically try to reconnect if the connection is lost
  err = db.pingAndSetupQueries()
  if err != nil {
    if retries <= 3 {  // Too many retries here could lead to stack overflow as a result of recursion
      // TODO: Implement no new trades flag
      push.Error(fmt.Sprintf("Database connection lost\n  -> Trying again in %d\n", *backoff_sec), err)
      log.Printf(
        "[ ERROR ]\tFailed to reconnect to database\n" +
        // TODO: Implement NO_NEW_TRADES
        "  -> Setting NO_NEW_TRADES == true\n" +
        "  -> Error: %s\n -> Retrying in %d seconds\n",
        err.Error(), *backoff_sec,
      )
      backoff.Backoff(backoff_sec)
      db.errorHandler(err, func_name, response, query, retries + 1, backoff_sec)
    } else {
      push.Error("MAX RETRIES REACHED.\nCLOSING ALL POSITIONS AND SHUTTING DOWN.", err)
      log.Printf("[ ERROR ]\tMAX RETRIES REACHED.\nCLOSING ALL POSITIONS AND SHUTTING DOWN.\n")
      order.CloseAllPositions(2, 0)
      // TODO: Think this should be log.Fatal
      log.Panicln("SHUTTING DOWN")
    }
  }
  // If connection is successful, try to execute the query again
  if retries > 3 {
    push.Error("MAX RETRIES REACHED.\nCLOSING ALL POSITIONS AND SHUTTING DOWN.", err)
    log.Printf("[ ERROR ]\tMAX RETRIES REACHED.\nCLOSING ALL POSITIONS AND SHUTTING DOWN.\n")
    order.CloseAllPositions(2, 0)
    // TODO: Think this should be log.Fatal
    log.Panicln("SHUTTING DOWN")
  }
  db.queryHandler(query, *backoff_sec, retries + 1)
  // This code is not reached if the insert fails
  if retries > 0 {
    push.Message("Query successful after: %s retries")
  }
  log.Printf("[ OK ]\tQuery executed after: %d retries\n", retries)
}


// TODO: Change queries so they are correct(but have to see how the database is structured first)
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
    query.TriggerTime,
    query.TriggerPrice,
    query.FillTime,
    query.FilledAvgPrice,
    query.OrderSentTime,
    query.BadForAnalysis,
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
    query.TriggerTime,
    query.TriggerPrice,
    query.FillTime,
    query.FilledAvgPrice,
    query.OrderSentTime,
    query.TrailingStop,
    query.BadForAnalysis,
  )
  if err != nil {
    db.errorHandler(err, "insertPosition", response, query, retries, &backoff_sec)
  }
}


func (db *Database) deletePosition(query *Query, backoff_sec int, retries int) {
  response, err := db.delete_position.Exec(query.Symbol, query.StratName)
  if err != nil {
    db.errorHandler(err, "deletePosition", response, query, retries, &backoff_sec)
  }
}


func (db *Database) updateTrailingStop(query *Query, backoff_sec int, retries int) {
  response, err := db.update_trailing_stop.Exec(query.Symbol, query.StratName, query.TrailingStopPrice)
  if err != nil {
    db.errorHandler(err, "updateTrailingStop", response, query, retries, &backoff_sec)
  }
}


func (db *Database) queryHandler(query *Query, backoff_sec int, retries int) {
  switch query.Action {
    case "open":
      db.insertTrade(query, backoff_sec, retries)
      db.insertPosition(query, backoff_sec, retries)
    case "close":
      db.insertPosition(query, backoff_sec, retries)
      db.deletePosition(query, backoff_sec, retries)
    case "update":
      db.updateTrailingStop(query, backoff_sec, retries)
    default:
      push.Error("Invalid query type", nil)
      log.Printf("[ ERROR ]\tInvalid query Action: %s\n", query.Action)
      // TODO: Should probably shut down here
  }
}


func (db *Database) listen() {
  for {
    query := <-db.db_chan
    db.queryHandler(query, 0, 0)
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


// Constructor
func NewDatabase(wg *sync.WaitGroup, db_chan chan *Query) {
  defer wg.Done()
  db := &Database{}
  db.connect()
  defer db.conn.Close()
  err := db.prepQueries()
  if err != nil {
    log.Panicf("[ ERROR ]\tsetupQueries() failed: %s\n", err.Error())
  }
  db.db_chan = db_chan
  db.listen()
  fmt.Println("After listen")
}
