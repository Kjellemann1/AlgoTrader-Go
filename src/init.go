
package src

import (
  "log"
  "os"
  "time"
  "io"
  "net/http"
  "github.com/joho/godotenv"
  "github.com/Kjellemann1/AlgoTrader-Go/src/constant"
)


// Set up logging to both console and file.
func init() {
  // Make log file with unique datetime name in UTC
  name := "logs/" + time.Now().In(time.UTC).Format(time.DateTime) + ".log"
  logfile, err := os.OpenFile(name, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)

  // Panic if log file creation fails
  if err != nil {
    panic(err)
  }

  multiWriter := io.MultiWriter(os.Stdout, logfile)
  log.SetOutput(multiWriter)
}


// Load the API key and secret from the .env file
func init() {
  err := godotenv.Load()
  if err != nil {
    panic(err)
  }

  constant.KEY = os.Getenv("PaperKey")
  constant.SECRET = os.Getenv("PaperSecret")

  // Panic if missing key or secret
  if constant.KEY == "" || constant.SECRET == "" {
    panic("Missing PaperKey or PaperSecret")
  }

  constant.AUTH_HEADERS = http.Header{
    "accept": {"application/json"},
    "APCA-API-KEY-ID": {constant.KEY},
    "APCA-API-SECRET-KEY": {constant.SECRET},
  }
}
