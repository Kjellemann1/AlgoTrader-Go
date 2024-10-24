
package src

import (
  "github.com/Kjellemann1/AlgoTrader-Go/src/indicator"
)


func (a *Asset) testingStrategy() {
  if _, ok := a.Positions["test"]; !ok {
    a.OpenPosition("long", "IOC", "test")
  } else {
    a.ClosePosition("IOC", "test")
  }
}

// TODO: This strat does not work until filling of windows with historical data at start is implemented
func (a *Asset) testingStrategy2() {
  rsi := indicator.RSI(a.Close, 14)
  if rsi[len(rsi)-1] > 70 {
    // a.ClosePosition("long", "IOC", "test_RSI")
  }
}
