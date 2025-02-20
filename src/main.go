package main

import (
  "os"
  "log"
  "fmt"
  "sync"
  "os/signal"
  "syscall"
  "context"
  "github.com/Kjellemann1/AlgoTrader-Go/constant"
  "github.com/Kjellemann1/AlgoTrader-Go/order"
)

var  globRwm sync.RWMutex

func prepAssetsMap() map[string]map[string]*Asset {
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

  return assets
}

func signalHandler(marketCancel context.CancelFunc, accountCancel context.CancelFunc, rootCancel context.CancelFunc, db_chan chan *Query) {
  sigChan := make(chan os.Signal, 1)
  defer close(sigChan)
  signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

  for {
    sig := <-sigChan
    NNP.NoNewPositionsTrue("Run")
    log.Printf("Received signal: %v\n", sig)
    fmt.Println("What do you want to do?")
    fmt.Printf("  -> 1) Abort\n  -> 2) Save state and shutdown\n  -> 3) Close all positions and shutdown\n")
    var input string
    fmt.Scanln(&input)

    switch input {
    case "1":
      NNP.NoNewPositionsFalse("Run")
      log.Println("Shutdown aborted. Resuming...")
      continue
    case "2":
      log.Println("Saving state and shutting down...")
      marketCancel()
      accountCancel()
      rootCancel()
      return
    case "3":
      fmt.Println("ARE YOU SURE YOU WANT TO CLOSE ALL POSITIONS? (y/n)")
      fmt.Scanln(&input)

      switch input {
      case "Y", "y":
        log.Println("Closing all positions and shutting down...")
        order.CloseAllPositions(5, 5)
        db_chan <- &Query{Action: "delete_all_positions"}
        marketCancel()
        accountCancel()
        rootCancel()
      case "N", "n":
        NNP.NoNewPositionsFalse("Run")
        log.Println("Shutdown aborted. Resuming...")
        continue
      default:
        log.Println("Invalid input")
        NNP.NoNewPositionsFalse("Run")
        log.Println("Shutdown aborted. Resuming...")
      }
    }
  }
}

func main() {
  fmt.Println("Starting AlgoTrader ...")

  rootCtx, rootCancel := context.WithCancel(context.Background())
  defer rootCancel()
  marketCtx, marketCancel := context.WithCancel(rootCtx)
  accountCtx, accountCancel := context.WithCancel(rootCtx)
  db_chan := make(chan *Query, len(constant.STOCK_LIST) + len(constant.CRYPTO_LIST))

  go signalHandler(marketCancel, accountCancel, rootCancel, db_chan)

  assets := prepAssetsMap()
  var wg sync.WaitGroup

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

  wg.Wait()
  close(db_chan)
}
