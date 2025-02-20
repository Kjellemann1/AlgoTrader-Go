package main

import (
  "os"
  "log"
  "fmt"
  "os/signal"
  "syscall"
  "context"
  "github.com/Kjellemann1/AlgoTrader-Go/order"
)

func shutdownSignalHandler(marketCancel context.CancelFunc, accountCancel context.CancelFunc, db_chan chan *Query) {
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
      default :
        NNP.NoNewPositionsFalse("Run")
        log.Println("Shutdown aborted. Resuming...")
      }
    }
  }
}
