
package src

import (
  "fmt"
  "sync"
  "log"
  "database/sql"
  "github.com/shopspring/decimal"
  _ "github.com/go-sql-driver/mysql"
  "github.com/Kjellemann1/AlgoTrader-Go/src/util/backoff"
  "github.com/Kjellemann1/AlgoTrader-Go/src/constant"
  "github.com/Kjellemann1/AlgoTrader-Go/src/util/push"
  "github.com/Kjellemann1/AlgoTrader-Go/src/order"
)


type Query struct {
  query_type       string
  price            float64
  db_key           string
  action           string
  order_id         string
  symbol           string
  asset_class      string
  side             string  // "buy", "sell"
  strat_name       string
  order_type       string
  qty              decimal.Decimal  // Stored as a string to avoid floating point errors
  price_time       string
  trigger_time     string
  trigger_price    float64
  fill_time        string
  filled_avg_price float64
  order_sent_time  string
  trailing_stop    float64
  bad_for_analysis bool
}


type Database struct {
  conn                 *sql.DB
  query_chan           chan *Query
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
      action, order_id, symbol, asset_class, side, strat_name, order_type, qty, price_time,
      trigger_time, trigger_price, fill_time, filled_avg_price, order_sent_time, bad_for_analysis
    ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);
  `)
  if err != nil {
    return err
  }
  db.insert_position, err = db.conn.Prepare(`
    INSERT INTO positions (
      db_key, order_id, symbol, asset_class, side, strat_name, order_type, qty, price_time,
      trigger_time, trigger_price, fill_time, filled_avg_price, order_sent_time, trailing_stop, bad_for_analysis
    ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);
  `)
  if err != nil {
    return err
  }
  db.delete_position, err = db.conn.Prepare(`
    DELETE FROM positions WHERE db_key = ?;
  `)
  if err != nil {
    return err
  }
  db.update_trailing_stop, err = db.conn.Prepare(`
    UPDATE positions SET trailing_stop = ? WHERE db_key = ?;
  `)
  if err != nil {
    return err
  }
  return nil
}


func (db *Database) pingAndsetupQueries() error {
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
  if err != nil {
    if retries == 0 {
      push.Warning("Failed to insert trade", err)
      log.Printf(
        "[ WARNING ]\t%s failed\n  -> Retries: %d\n  -> Error: %s\n  -> Response: %s\n",
        func_name, retries, err.Error(), response,
      )
    } else {
      push.Error("Failed to insert trade on retry", err)
      log.Printf(
        "[ ERROR ]\tinsert_trade.Exec() failed\n  -> Retries: %d\n  -> Error: %s\n",
        retries, err.Error(),
      )
    }
    // Ping database. Ping() should automatically try to reconnect if the connection is lost
    err = db.pingAndsetupQueries()
    if err != nil {
      if retries <= 3 {  // Too many retries here could lead to stack overflow as a result of recursion
        // TODO: Implement no new trades flag
        push.Error(fmt.Sprintf("Database connection lost\n  -> Trying again in %d\n", *backoff_sec), err)
        log.Printf(
          "[ ERROR ]\tFailed to reconnect to database\n" +
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
        log.Panicln("SHUTTING DOWN")
      }
    }
  }
  // If connection is successful, try to insert the trade again
  db.queryHandler(query, *backoff_sec, retries + 1)
  // This code is not reached if the insert fails
  if retries > 0 {
    push.Message("Trade inserted after: %s retries")
  }
  log.Printf("[ OK ]\tTrade inserted after: %d retries\n", retries)
}


// TODO: Change queries so they are correct(but have to see how the database is structured first)
func (db *Database) insertTrade(query *Query, backoff_sec int, retries int) {
  response, err := db.insert_trade.Exec(
    query.action,
    query.order_id,
    query.symbol,
    query.asset_class,
    query.side,
    query.strat_name,
    query.order_type,
    query.qty, 
    query.price_time,
    query.trigger_time,
    query.trigger_price,
    query.fill_time,
    query.filled_avg_price,
    query.order_sent_time,
    query.bad_for_analysis,
  )
  db.errorHandler(err, "insertTrade", response, query, retries, &backoff_sec)
}


func (db *Database) insertPosition(query *Query, backoff_sec int, retries int) {
  response, err := db.insert_position.Exec(
    query.db_key,
    query.order_id,
    query.symbol,
    query.asset_class,
    query.side,
    query.strat_name,
    query.order_type,
    query.qty,
    query.price_time,
    query.trigger_time,
    query.trigger_price,
    query.fill_time,
    query.filled_avg_price,
    query.order_sent_time,
    query.trailing_stop,
    query.bad_for_analysis,
  )
  db.errorHandler(err, "insertPosition", response, query, retries, &backoff_sec)
}


func (db *Database) deletePosition(query *Query, backoff_sec int, retries int) {
  response, err := db.delete_position.Exec(query.db_key)
  db.errorHandler(err, "deletePosition", response, query, retries, &backoff_sec)
}


func (db *Database) updateTrailingStop(query *Query, backoff_sec int, retries int) {
  response, err := db.update_trailing_stop.Exec(query.db_key, query.price)
  db.errorHandler(err, "updateTrailingStop", response, query, retries, &backoff_sec)
}


func (db *Database) queryHandler(query *Query, backoff_sec int, retries int) {
  switch query.query_type {
    case "trade":
      db.insertTrade(query, backoff_sec, retries)
    case "insert_position":
      db.insertPosition(query, backoff_sec, retries)
    case "delete_position":
      db.deletePosition(query, backoff_sec, retries)
    case "update_trailing_stop":
      db.updateTrailingStop(query, backoff_sec, retries)
    default:
      push.Error("Invalid query type", nil)
      log.Printf("[ ERROR ]\tInvalid query type: %s\n", query.query_type)
  }
}


func (db *Database) listen() {
  for {
    query := <-db.query_chan
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
func NewDatabase(wg *sync.WaitGroup) {
  defer wg.Done()
  db := &Database{}
  db.connect()
  defer db.conn.Close()
  err := db.prepQueries()
  if err != nil {
    log.Panicf("[ ERROR ]\tsetupQueries() failed: %s\n", err.Error())
  }
  db.listen()
}
