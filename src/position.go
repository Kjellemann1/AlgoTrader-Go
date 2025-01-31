
// Position objects are stored in the Asset object of the given asset.
// Each asset can have multiple positions, but only one position per strategy.

package src

import (
  "time"
  "log"
  "github.com/shopspring/decimal"
)


type Position struct {
  Symbol                 string
  AssetClass             string
  StratName              string
  Qty                    decimal.Decimal
  BadForAnalysis         bool
  PositionID             string  // The position id is primarily used to track the order status. 
                                 // It contains the strategy name, which is used to identify the strategy that
                                 // placed the order when getting order updates from the broker.
  OpenOrderPending       bool
  OpenTriggerTime        time.Time 
  OpenSide               string
  OpenOrderTimeBefore    time.Time 
  OpenOrderTimeAfter     time.Time
  OpenOrderType          string
  OpenTriggerPrice       float64
  OpenPriceTime          time.Time
  OpenPriceReceivedTime  time.Time
  OpenFillTime           time.Time
  OpenFilledAvgPrice     float64

  CloseOrderPending      bool
  CloseOrderTimeBefore   time.Time 
  CloseOrderTimeAfter    time.Time
  CloseOrderType         string
  CloseFilledQty         decimal.Decimal  // What is this used for? Redundant? Might be for partial closing of positions
  CloseTriggerTime       time.Time
  CloseTriggerPrice      float64
  CloseFillTime          time.Time
  CloseFilledAvgPrice    float64
  ClosePriceTime         time.Time
  ClosePriceReceivedTime time.Time
}


// Constructor for Position
func NewPosition(symbol string, price float64) *Position {
  trigger_time := time.Now().UTC()
  p := &Position{
    BadForAnalysis: false,
    OpenOrderPending: true,
    CloseOrderPending: false,
    OpenTriggerTime: trigger_time,
    OpenTriggerPrice: price,
  }
  return p
}


func (p *Position) LogOpen(strat_name string) *Query {
  log.Println("OPEN\t" + p.Symbol + "\t" + strat_name)
  query := &Query{
    Action: "open",
    PositionID: p.PositionID,
    Symbol: p.Symbol,
    AssetClass: p.AssetClass,
    Side: p.OpenSide,
    StratName: p.StratName,
    OrderType: p.OpenOrderType,
    Qty: p.Qty,
    PriceTime: p.OpenPriceTime,
    ReceivedTime: p.OpenPriceReceivedTime,
    TriggerTime: p.OpenTriggerTime,
    TriggerPrice: p.OpenTriggerPrice,
    FillTime: p.OpenFillTime,
    FilledAvgPrice: p.OpenFilledAvgPrice,
    OrderTimeBefore: p.OpenOrderTimeBefore,
    OrderTimeAfter: p.OpenOrderTimeAfter,
    BadForAnalysis: p.BadForAnalysis,
    TrailingStopPrice: 0.0, // TODO: Change this when trailing stop is implemented
  }
  // TODO: Implement log if buffer is full
  return query
}


func (p *Position) LogClose(strat_name string) *Query {
  log.Println("CLOSE\t" + p.Symbol + "\t" + strat_name)
  var side string
  if p.OpenSide == "long" {
    side = "sell"
  } else {
    side = "buy"
  }
  query := &Query{
    Action: "close",
    PositionID: p.PositionID,
    Symbol: p.Symbol,
    AssetClass: p.AssetClass,
    Side: side,
    StratName: p.StratName,
    OrderType: p.CloseOrderType,
    Qty: p.Qty,
    PriceTime: p.ClosePriceTime,
    ReceivedTime: p.ClosePriceReceivedTime,
    TriggerTime: p.CloseTriggerTime,
    TriggerPrice: p.CloseTriggerPrice,
    FillTime: p.CloseFillTime,
    FilledAvgPrice: p.CloseFilledAvgPrice,
    OrderTimeBefore: p.CloseOrderTimeBefore,
    OrderTimeAfter: p.CloseOrderTimeAfter,
    BadForAnalysis: p.BadForAnalysis,
  }
  // TODO: Implement log if buffer is full
  return query
}
