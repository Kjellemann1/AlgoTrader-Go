
package order

import (
  "fmt"
  "log"
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
    log.Println("Error sending order in order.OpenLongIOC():", err.Error())
    return err
  }
  return nil
}
