
package src

import (
  "sync"

  "github.com/Kjellemann1/AlgoTrader-Go/src/constant"
)

var rwmu sync.RWMutex

var NO_NEW_TRADES bool = false


func Run() {
  query_chan := make(chan string, constant.CHANNEL_BUFFER_SIZE)
  order_update_chan := make(map[string]chan OrderUpdate)

  var wg sync.WaitGroup

  // Database
  wg.Add(1)
  go NewDatabase(&wg)

  // Account
  wg.Add(1)
  go NewAccount(order_update_chan, &wg)

  // Stocks
  if len(constant.STOCK_LIST) > 0 {
    stock_asset_map := map[string]*Asset{}
    order_update_chan["stock"] = make(chan OrderUpdate, constant.CHANNEL_BUFFER_SIZE)
    for _, symbol := range constant.STOCK_LIST {
      stock_asset_map[symbol] = NewAsset()
    }
    wg.Add(1)
    go NewMarket(
      "stock", "wss://stream.data.alpaca.markets/v2/iex", 
      stock_asset_map, query_chan, order_update_chan["stock"], &wg,
    )
  }

  // Crypto
  if len(constant.CRYPTO_LIST) > 0 {
    crypto_asset_map := make(map[string]*Asset)
    order_update_chan["crypto"] = make(chan OrderUpdate, constant.CHANNEL_BUFFER_SIZE)
    for _, symbol := range constant.CRYPTO_LIST {
      crypto_asset_map[symbol] = NewAsset()
    }
    wg.Add(1)
    go NewMarket(
      "crypto", "wss://stream.data.alpaca.markets/v1beta3/crypto/us",
      crypto_asset_map, query_chan, order_update_chan["crypto"], &wg,
    )
  } 

  wg.Wait()

  close(query_chan)
  close(order_update_chan["stock"])
  close(order_update_chan["crypto"])
}
