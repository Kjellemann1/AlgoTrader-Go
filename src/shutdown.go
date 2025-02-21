package main

import (
  "os"
  "log"
  "fmt"
  "time"
  "os/signal"
  "syscall"
  "context"
  "github.com/Kjellemann1/AlgoTrader-Go/request"
)

func stallIfOrdersPending(assets map[string]map[string]*Asset) {
  ticker := time.NewTicker(3 * time.Second)
  defer ticker.Stop()
  for range ticker.C {
    pending := pendingOrders(assets)
    if len(pending) == 0 {
      return
    } else {
      log.Println("Waiting for pending orders:")
      for _, v := range pending {
        for symbol := range v {
          log.Println("  ->", symbol)
        }
      }
    }
  }
}

func shutdownHandler(marketCancel context.CancelFunc, accountCancel context.CancelFunc, assets map[string]map[string]*Asset, db_chan chan *Query) {
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
      stallIfOrdersPending(assets)
      accountCancel()
      db_chan <- nil
      return
    case "3":
      fmt.Println("ARE YOU SURE YOU WANT TO CLOSE ALL POSITIONS? (y/n)")
      fmt.Scanln(&input)

      switch input {
      case "Y", "y":
        log.Println("Closing all positions and shutting down...")
        request.CloseAllPositions(2, 0)
        marketCancel()
        accountCancel()
        fmt.Println("Do you want to clear the positions table? (y/n)")
        fmt.Scanln(&input)
        if input == "Y" || input == "y" {
          db_chan <- &Query{Action: "delete_all_positions"}
        }
        db_chan <- nil
        return
      default :
        NNP.NoNewPositionsFalse("Run")
        log.Println("Shutdown aborted. Resuming...")
      }
    }
  }
}
