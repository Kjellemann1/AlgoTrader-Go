
package src

import (
  "sync"

  "github.com/Kjellemann1/AlgoTrader-Go/src/constant"
)


// This is for all intents and purposes the main function
func Run() {
  db_chan := make(chan *Query, len(constant.STOCK_LIST) + len(constant.CRYPTO_LIST))

  assets := make(map[string]map[string]*Asset)
  if len(constant.STOCK_LIST) > 0 {
    assets["stock"] = make(map[string]*Asset)
    for _, symbol := range constant.STOCK_LIST {
      assets["stock"][symbol] = NewAsset("stock", symbol)
    }
    GetHistBars(assets["stock"], "stock")
  }
  if len(constant.CRYPTO_LIST) > 0 {
    assets["crypto"] = make(map[string]*Asset)
    for _, symbol := range constant.CRYPTO_LIST {
      assets["crypto"][symbol] = NewAsset("crypto", symbol)
    }
    GetHistBars(assets["crypto"], "crypto")
  }

  var wg sync.WaitGroup


  // Database
  wg.Add(1)
  db := NewDatabase(db_chan)
  go db.Start(&wg)

  // Account
  wg.Add(1)
  a := NewAccount(assets, db_chan)
  go a.Start(&wg)

  // Stock
  if _, ok := assets["stock"]; ok {
    stockMarket := NewMarket("stock", constant.WSS_STOCK, assets["stock"])
    wg.Add(1)
    go stockMarket.Start(&wg)
  }

  // Crypto
  if _, ok := assets["crypto"]; ok {
    cryptoMarket := NewMarket("crypto", constant.WSS_CRYPTO, assets["crypto"])
    wg.Add(1)
    go cryptoMarket.Start(&wg)
  } 

  wg.Wait()
  close(db_chan)
}
