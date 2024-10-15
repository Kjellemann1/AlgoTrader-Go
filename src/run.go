
package src

import (
  "os"
  "net/http"
  "sync"

  "github.com/joho/godotenv"
)

var AUTH_HEADERS http.Header

func load_env() {
  err := godotenv.Load()
  if err != nil {
    panic(err)
  }
  AUTH_HEADERS = http.Header{
    "accept": {"application/json"},
    "APCA-API-KEY-ID": {os.Getenv("PaperKey")},
    "APCA-API-SECRET-KEY": {os.Getenv("PaperSecret")},
  }
}

func socket(asset_class string, url string, query_chan chan string, position_monitor_chan chan map[string]string, wg *sync.WaitGroup) {
  defer wg.Done()
  stock_socket := Market{
    asset_class: asset_class,
    query_chan: query_chan,
    position_monitor_chan: position_monitor_chan,
    url: url,
  }
  stock_socket.Init()
}

// This is in practice the main function call
func Run() {
  load_env()

  // Initialiser kanalene her
  query_chan := make(chan string)
  stock_position_monitor_chan := make(chan map[string]string)
  crypto_position_monitor_chan := make(chan map[string]string)

  var wg sync.WaitGroup
  wg.Add(1)

  // Start socket-goroutines og send spesifikke kanaler til hver socket
  go socket("stock", "wss://stream.data.alpaca.markets/v2/test", query_chan, stock_position_monitor_chan, &wg)
  // go socket("crypto", "wss://stream.data.alpaca.markets/v2/test", query_chan, crypto_position_monitor_chan, &wg)

  wg.Wait()

  // Lukk kanalene hvis de ikke skal brukes mer
  close(query_chan)
  close(stock_position_monitor_chan)
  close(crypto_position_monitor_chan)
}
