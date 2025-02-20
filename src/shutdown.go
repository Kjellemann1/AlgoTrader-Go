package main

import (
  "os"
  "log"
  "fmt"
  "time"
  "os/signal"
  "syscall"
  "context"
  "github.com/Kjellemann1/AlgoTrader-Go/order"
)

func ordersPending(assets map[string]map[string]*Asset) bool {
  for _, class := range assets {
    for _, asset := range class {
      for _, position := range (*asset).Positions {
        if position.OpenOrderPending || position.CloseOrderPending {
          return true
        }
      }
    }
  }
  return false
}

func checkOrdersPending(assets map[string]map[string]*Asset) {
  ticker := time.NewTicker(5 * time.Second)
  defer ticker.Stop()
  for range ticker.C {
    if !ordersPending(assets) {
      return
    }
  }
}

func shutdownSignalHandler(marketCancel context.CancelFunc, accountCancel context.CancelFunc, assets map[string]map[string]*Asset, db_chan chan *Query) {
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
      checkOrdersPending(assets)
      accountCancel()
      db_chan <- nil
      return
    case "3":
      fmt.Println("ARE YOU SURE YOU WANT TO CLOSE ALL POSITIONS? (y/n)")
      fmt.Scanln(&input)

      switch input {
      case "Y", "y":
        log.Println("Closing all positions and shutting down...")
        // TODO: Check if remove open orders is necessary
        order.CloseAllPositions(5, 5)
        marketCancel()
        accountCancel()
        db_chan <- &Query{Action: "delete_all_positions"}
        db_chan <- nil
      default :
        NNP.NoNewPositionsFalse("Run")
        log.Println("Shutdown aborted. Resuming...")
      }
    }
  }
}
