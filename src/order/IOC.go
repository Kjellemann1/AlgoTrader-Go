
package order

import (
  "fmt"
  "log"
  "github.com/shopspring/decimal"
)


func OpenLongIOC(symbol string, order_id string, last_price float64) error {
  qty := CalculateOpenQty("stock", last_price)
  payload := fmt.Sprintf(`{
    "symbol": "%s",
    "client_order_id": "%s",
    "qty": "%s",
    "side": "buy",
    "type": "market",
    "time_in_force": "ioc",
    "order_class": "simple"
  }`, symbol, order_id, qty)
  if err := SendOrder(payload); err != nil {
    log.Println("Error sending order in order.CloseLongIOC():", err.Error())
    return err
  }
  return nil
}


func CloseIOC(side string, symbol string, order_id string, qty decimal.Decimal) error {
  fmt.Println("IOC 31")
  payload := fmt.Sprintf(`{
    "symbol": "%s",
    "client_order_id": "%s",
    "qty": "%s",
    "side": "%s",
    "type": "market",
    "time_in_force": "ioc",
    "order_class": "simple"
  }`, symbol, fmt.Sprintf("%s_close", order_id), qty, side)
  if err := SendOrder(payload); err != nil {
    log.Println("Error sending order in order.CloseIOC():", err.Error())
    return err
  }
  return nil
}
