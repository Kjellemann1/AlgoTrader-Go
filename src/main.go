package main

import (
  "log"
  "sync"
  "context"
  "github.com/Kjellemann1/AlgoTrader-Go/constant"
)

var globRwm sync.RWMutex

func main() {
  log.Println("Starting AlgoTrader ...")

  rootCtx, rootCancel := context.WithCancel(context.Background())
  defer rootCancel()
  marketCtx, marketCancel := context.WithCancel(rootCtx)
  accountCtx, accountCancel := context.WithCancel(rootCtx)

  db_chan := make(chan *Query, len(constant.STOCK_SYMBOLS) + len(constant.CRYPTO_SYMBOLS))
  defer close(db_chan)

  var wg sync.WaitGroup
  defer wg.Wait()

  assets := prepAssetsMap()
  fillRollingWindows(assets)

  wg.Add(1)
  go shutdownHandler(&wg, marketCancel, accountCancel, assets, db_chan)

  wg.Add(1)
  db := NewDatabase(db_chan, assets)
  go db.start(&wg)

  wg.Add(1)
  a := NewAccount(assets, constant.WSS_ACCOUNT, db_chan)

  go a.start(&wg, accountCtx, 2)

  if _, ok := assets["stock"]; ok {
    sm := NewMarket("stock", constant.WSS_STOCK, assets["stock"])
    wg.Add(1)
    go sm.start(&wg, marketCtx, 2)
  }

  if _, ok := assets["crypto"]; ok {
    cm := NewMarket("crypto", constant.WSS_CRYPTO, assets["crypto"])
    wg.Add(1)
    go cm.start(&wg, marketCtx, 2)
  } 
}
