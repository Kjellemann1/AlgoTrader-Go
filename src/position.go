
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
  OrderID               string  // The order id is primarily used to track the order status. It contains the strategy name, which is used to
                                // identify the strategy that placed the order when getting order updates from the broker.
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
  CloseOrderSentTime    string
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
    OpenPriceTime: time.Now().UTC().Format(time.RFC3339),
  }
  p.Init()
  return p
}

// Initializes the Position
func (p *Position) Init() {
  // TODO: Implement this
}

// Log the opening to database and console
func (p *Position) LogOpen() {
  // TODO
  // Implement log if buffer is full
}

// Log the closing to database and console
func (p *Position) LogClose() {
  // TODO
  // Implement log if buffer is full
}
