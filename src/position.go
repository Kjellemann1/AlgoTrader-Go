
// Position objects are stored in the Asset object of the given asset.
// Each asset can have multiple positions, but only one position per strategy.

package src

import (
  // Standard packages
  "time"

  // External packages
  "github.com/shopspring/decimal" // https://pkg.go.dev/github.com/shopspring/decimal#section-readme
)

type Position struct {
  Symbol                string
  AssetClass            string
  StratName             string
  Qty                   decimal.Decimal
  BadForAnalysis        bool
  OrderID               string  // The order id is primarily used to track the order status. It contains the strategy name, which is used to
                                // identify the strategy that placed the order when getting order updates from the broker.
  OpenSide              string
  OpenOrderPendingFlag  bool
  OpenOrderSentTime     time.Time
  OpenOrderType         string
  OpenTriggerTime       time.Time
  OpenTriggerPrice      float64
  OpenFillTime          time.Time
  OpenFilledQty         decimal.Decimal
  OpenFilledAvgPrice    float64
  OpenPriceTime         time.Time

  CloseOrderPendingFlag bool
  CloseOrderSentTime    time.Time
  CloseOrderType        string
  CloseFilledQty        decimal.Decimal
  CloseTriggerTime      time.Time
  CloseTriggerPrice     float64
  CloseFillTime         time.Time
  CloseFilledAvgPrice   float64
  ClosePriceTime        time.Time
}

// Constructor for Position
func NewPosition() *Position {
  p := &Position{
    BadForAnalysis: false,
    OpenOrderPendingFlag: false,
    CloseOrderPendingFlag: false,
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
