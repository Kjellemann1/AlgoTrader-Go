// Position objects are stored in the Asset object of the given asset.
// Each asset can have multiple positions, but only one position per strategy.

package main

import (
  "time"
  "sync"
  "github.com/shopspring/decimal"
  "github.com/Kjellemann1/AlgoTrader-Go/util"
)

type Position struct {
  Symbol                 string
  AssetClass             string
  StratName              string
  Qty                    decimal.Decimal
  BadForAnalysis         bool
  PositionID             string  // PositionID is primarily used to tie order updates to the correct
                                 // position object. It contains the strategy name which is grepped from 
                                 // "client_order_id" in order updates. The strategy name in combination
                                 // with the symbol is unique to the position.
  OpenOrderPending       bool
  OpenTriggerTime        time.Time 
  OpenSide               string
  OpenOrderType          string
  OpenTriggerPrice       float64
  OpenPriceTime          time.Time
  OpenPriceReceivedTime  time.Time
  OpenFillTime           time.Time
  OpenFilledAvgPrice     float64

  CloseOrderPending      bool
  CloseOrderType         string
  CloseTriggerTime       time.Time
  CloseTriggerPrice      float64
  CloseFillTime          time.Time
  CloseFilledAvgPrice    float64
  ClosePriceTime         time.Time
  ClosePriceReceivedTime time.Time

  NCloseOrders           int8
  TrailingStopBase       float64

  Rwm                    sync.RWMutex
}

func NewPosition(symbol string) *Position {
  zero, _ := decimal.NewFromString("0")

  return &Position{
    BadForAnalysis: false,
    OpenOrderPending: true,
    CloseOrderPending: false,
    Qty: zero,
  }
}

func checkTimeNil(t time.Time) *time.Time {
  if t.IsZero() {
    return nil
  } else {
    return &t
  }
}

func (p *Position) LogOpen() *Query {
  util.Open(util.AddWhitespace(p.Symbol, 10) + "\t" + p.StratName)

  return &Query{
    Action: "open",
    PositionID: p.PositionID,
    Symbol: p.Symbol,
    AssetClass: p.AssetClass,
    Side: p.OpenSide,
    StratName: p.StratName,
    OrderType: p.OpenOrderType,
    Qty: p.Qty,
    PriceTime: checkTimeNil(p.OpenPriceTime),
    ReceivedTime: checkTimeNil(p.OpenPriceReceivedTime),
    TriggerTime: checkTimeNil(p.OpenTriggerTime),
    FillTime: checkTimeNil(p.OpenFillTime),
    TriggerPrice: p.OpenTriggerPrice,
    FilledAvgPrice: p.OpenFilledAvgPrice,
    BadForAnalysis: p.BadForAnalysis,
  }
}

func (p *Position) LogClose() *Query {
  util.Close(util.AddWhitespace(p.Symbol, 10) + "\t" + p.StratName)

  var side string
  if p.OpenSide == "long" {
    side = "sell"
  } else {
    side = "buy"
  }

  p.NCloseOrders++

  return &Query{
    Action: "close",
    PositionID: p.PositionID,
    Symbol: p.Symbol,
    AssetClass: p.AssetClass,
    Side: side,
    StratName: p.StratName,
    OrderType: p.CloseOrderType,
    Qty: p.Qty,
    PriceTime: checkTimeNil(p.ClosePriceTime),
    ReceivedTime: checkTimeNil(p.ClosePriceReceivedTime),
    TriggerTime: checkTimeNil(p.CloseTriggerTime),
    FillTime: checkTimeNil(p.CloseFillTime),
    TriggerPrice: p.CloseTriggerPrice,
    FilledAvgPrice: p.CloseFilledAvgPrice,
    BadForAnalysis: p.BadForAnalysis,
    NCloseOrders: p.NCloseOrders,
  }
}
