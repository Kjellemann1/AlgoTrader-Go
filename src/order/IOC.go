
package order

import (
  "log"
  "github.com/shopspring/decimal"
)


func OpenLongIOC(symbol string, order_id string, last_price float64) error {
  qty := CalculateOpenQty("stock", last_price)
  payload := `{` +
    `"symbol": "` + symbol + `", ` +
    `"client_order_id": "` + order_id + `", ` +
    `"qty": "` + qty.String() + `", ` +
    `"side": "buy", "type": "market", "time_in_force": "ioc", "order_class": "simple"` +
  `}`
  if err := SendOrder(payload); err != nil {
    log.Println("Error sending order in order.OpenLongIOC():", err.Error())
    return err
  }
  return nil
}


// TODO: Check if position exists if order fails, and implement retry with backoff.
func CloseIOC(side string, symbol string, order_id string, qty decimal.Decimal) error {
  payload := `{` +
    `"symbol": "` + symbol + `", ` +
    `"client_order_id": "` + order_id + `_close", ` +
    `"qty": "` + qty.String() + `", ` +
    `"side": "` + side + `", ` +
    `"type": "market", "time_in_force": "ioc", "order_class": "simple"` +
  `}`
  if err := SendOrder(payload); err != nil {
    log.Println("Error sending order in order.CloseIOC():", err.Error())
    return err
  }
  return nil
}
