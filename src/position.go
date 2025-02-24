// Position objects are stored in the Asset object of the given asset.
// Each asset can have multiple positions, but only one position per strategy.

package main

import (
  "time"
  "log"
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

  Rwm                    sync.RWMutex
}

func NewPosition(symbol string) *Position {
  zero, _ := decimal.NewFromString("0")
  p := &Position{
    BadForAnalysis: false,
    OpenOrderPending: true,
    CloseOrderPending: false,
    Qty: zero,
  }
  return p
}

func (p *Position) LogOpen() *Query {
  log.Println("OPEN >>\t" + util.AddWhitespace(p.Symbol, 10) + "\t" + p.StratName)

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
    BadForAnalysis: p.BadForAnalysis,
  }

  return query
}

func (p *Position) LogClose() *Query {
  log.Println("<< CLOSE\t" + util.AddWhitespace(p.Symbol, 10) + "\t" + p.StratName)

  var side string
  if p.OpenSide == "long" {
    side = "sell"
  } else {
    side = "buy"
  }

  p.NCloseOrders++

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
    BadForAnalysis: p.BadForAnalysis,
    NCloseOrders: p.NCloseOrders,
  }

  return query
}
