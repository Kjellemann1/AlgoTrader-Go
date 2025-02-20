package main

import (
  "log"
  "sync"
  "context"
  "github.com/Kjellemann1/AlgoTrader-Go/constant"
)

var  globRwm sync.RWMutex

func main() {
  log.Println("Starting AlgoTrader ...")

  rootCtx, rootCancel := context.WithCancel(context.Background())
  defer rootCancel()
  marketCtx, marketCancel := context.WithCancel(rootCtx)
  accountCtx, accountCancel := context.WithCancel(rootCtx)

  db_chan := make(chan *Query, len(constant.STOCK_LIST) + len(constant.CRYPTO_LIST))
  defer close(db_chan)

  go shutdownSignalHandler(marketCancel, accountCancel, db_chan)

  var wg sync.WaitGroup
  defer wg.Wait()

  assets := prepAssetsMap()
  fillHistBars(assets)

  wg.Add(1)
  db := NewDatabase(db_chan)
  go db.Start(&wg, assets)

  wg.Add(1)
  a := NewAccount(assets, db_chan)
  go a.Start(&wg, accountCtx)

  if _, ok := assets["stock"]; ok {
    stockMarket := NewMarket("stock", constant.WSS_STOCK, assets["stock"])
    wg.Add(1)
    go stockMarket.Start(&wg, marketCtx)
  }

  if _, ok := assets["crypto"]; ok {
    cryptoMarket := NewMarket("crypto", constant.WSS_CRYPTO, assets["crypto"])
    wg.Add(1)
    go cryptoMarket.Start(&wg, marketCtx)
  } 
}
