
package src

import (
  "sync"
  "fmt"
  "github.com/Kjellemann1/AlgoTrader-Go/src/constant"
)

var rwmu sync.RWMutex


func Run() {

  db_chan := make(chan *Query, constant.CHANNEL_BUFFER_SIZE)

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

  fmt.Println("Close:", assets["crypto"]["BTC/USD"].Close)
  fmt.Println("Close:", assets["crypto"]["LTC/USD"].Close)
  fmt.Println("Close:", assets["crypto"]["ETH/USD"].Close)
  fmt.Println("Close:", assets["crypto"]["SHIB/USD"].Close)
  fmt.Println("Close:", assets["crypto"]["AAVE/USD"].Close)
  fmt.Println("Close:", assets["crypto"]["AVAX/USD"].Close)
  fmt.Println("Close:", assets["crypto"]["BAT/USD"].Close)
  fmt.Println("Close:", assets["crypto"]["BCH/USD"].Close)


  panic("")
  var wg sync.WaitGroup

  // Database
  wg.Add(1)
  go NewDatabase(&wg, db_chan)

  // Account)
  wg.Add(1)
  go NewAccount(&wg, assets, db_chan)

  // Stock
  if _, ok := assets["stock"]; ok {
    wg.Add(1)
    go NewMarket(
      "stock", "wss://stream.data.alpaca.markets/v2/iex", 
      assets["stock"], &wg,
    )
  }

  // Crypto
  if _, ok := assets["crypto"]; ok {
    wg.Add(1)
    go NewMarket(
      "crypto", "wss://stream.data.alpaca.markets/v1beta3/crypto/us",
      assets["crypto"], &wg,
    )
  } 

  wg.Wait()
  close(db_chan)
}
