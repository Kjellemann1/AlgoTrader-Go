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
  var retries int
  ticker := time.NewTicker(3 * time.Second)
  defer ticker.Stop()
  for range ticker.C {
    pending := pendingOrders(assets)
    if len(pending) == 0 {
      return
    } else if retries > 10 {
      log.Printf("[ WARNING ]\tTimeout (30 seconds) waiting for pending orders. Shutting down ...\n")
      return
    } else {
      retries++
      log.Println("Waiting for pending orders:")
      for symbol, positions := range pending {
        count_open := 0
        count_close := 0
        for _, pos := range positions {
          if pos.OpenOrderPending {
            count_open++
          } else {
            count_close++
          }
        }
        log.Printf("  -> %s: %d open, %d close\n", symbol, count_open, count_close)
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
    _, _ = fmt.Scanln(&input)

    switch input {
    case "1":
      NNP.NoNewPositionsFalse("Run")
      log.Println("Shutdown aborted. Resuming ...")
      continue
    case "2":
      log.Println("Saving state and shutting down ...")
      marketCancel()
      stallIfOrdersPending(assets)
      accountCancel()
      db_chan <- nil
      return
    case "3":
      fmt.Println("ARE YOU SURE YOU WANT TO CLOSE ALL POSITIONS? (y/n)")
      _, _ = fmt.Scanln(&input)

      switch input {
      case "Y", "y":
        log.Println("Closing all positions and shutting down...")
        request.CloseAllPositions(2, 0)
        marketCancel()
        accountCancel()
        fmt.Println("Do you want to clear the positions table? (y/n)")
        _, _ = fmt.Scanln(&input)
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
