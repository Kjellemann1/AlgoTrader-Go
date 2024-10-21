
// Position objects are stored in the Asset object of the given asset.
// Each asset can have multiple positions, but only one position per strategy.

package src

import (
  "time"
  "github.com/shopspring/decimal"
)


type Position struct {
  Symbol                string
  AssetClass            string
  StratName             string
  Qty                   decimal.Decimal
  BadForAnalysis        bool
  PositionID            string  // The position id is primarily used to track the order status. 
                                // It contains the strategy name, which is used to identify the strategy that
                                // placed the order when getting order updates from the broker.
  OpenOrderPending      bool
  OpenTriggerTime       time.Time 
  OpenSide              string
  OpenOrderSentTime     time.Time 
  OpenOrderType         string
  OpenTriggerPrice      float64
  OpenPriceTime         string
  OpenFillTime          string
  OpenFilledQty         decimal.Decimal  // What is this used for? Redundant?
  OpenFilledAvgPrice    float64

  CloseOrderPending     bool
  CloseOrderSentTime    time.Time
  CloseOrderType        string
  CloseFilledQty        decimal.Decimal  // What is this used for? Redundant?
  CloseTriggerTime      time.Time
  CloseTriggerPrice     float64
  CloseFillTime         string
  CloseFilledAvgPrice   float64
  ClosePriceTime        string
}


// Constructor for Position
func NewPosition(symbol string, price float64) *Position {
  p := &Position{
    BadForAnalysis: false,
    OpenOrderPending: true,
    CloseOrderPending: false,
    OpenTriggerTime: time.Now().UTC(),
    OpenTriggerPrice: price,
  }
  return p
}

func (p *Position) LogOpen(strat_name string) *Query {
  // TODO: Allocate on heap or stack?
  query := &Query{
    Action: "open",
    PositionID: p.PositionID,
    Symbol: p.Symbol,
    AssetClass: p.AssetClass,
    Side: p.OpenSide,
    StratName: p.StratName,
    OrderType: p.OpenOrderType,
    Qty: p.Qty,
    PriceTimeString: p.OpenPriceTime,
    TriggerTime: p.OpenTriggerTime,
    TriggerPrice: p.OpenTriggerPrice,
    FillTimeString: p.OpenFillTime,
    FilledAvgPrice: p.OpenFilledAvgPrice,
    OrderSentTime: p.OpenOrderSentTime,
    BadForAnalysis: p.BadForAnalysis,
    TrailingStopPrice: 0.0, // TODO: Change this when trailing stop is implemented
  }
  // TODO: Implement log if buffer is full
  return query
}

func (p *Position) LogClose(strat_name string) *Query {
  // TODO: Allocate on heap or stack?
  var side string
  if p.OpenSide == "buy" {
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
    PriceTimeString: p.ClosePriceTime,
    TriggerTime: p.CloseTriggerTime,
    TriggerPrice: p.CloseTriggerPrice,
    FillTimeString: p.CloseFillTime,
    FilledAvgPrice: p.CloseFilledAvgPrice,
    OrderSentTime: p.CloseOrderSentTime,
    BadForAnalysis: p.BadForAnalysis,
  }
  // TODO: Implement log if buffer is full
  return query
}
