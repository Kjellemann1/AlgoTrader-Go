package main

import (
  "log"
  "os"
  "time"
  "io"
  "net/http"
  "github.com/joho/godotenv"
  "github.com/Kjellemann1/AlgoTrader-Go/constant"
)

func init() {
  err := godotenv.Load()
  if err != nil {
    log.Panicln(err)
  }
}

func init() {
  name := os.Getenv("LogPath") + time.Now().In(time.UTC).Format(time.DateTime) + ".log"
  logfile, err := os.OpenFile(name, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
  if err != nil {
    log.Panicln(err)
  }
  multiWriter := io.MultiWriter(os.Stdout, logfile)
  log.SetOutput(multiWriter)
}

func init() {
  constant.PUSH_TOKEN = os.Getenv("PushoverToken")
  constant.PUSH_USER = os.Getenv("PushoverUser")

  constant.KEY = os.Getenv("PaperKey")
  constant.SECRET = os.Getenv("PaperSecret")

  constant.DB_USER = os.Getenv("DBUser")
  constant.DB_PASSWORD = os.Getenv("DBPassword")
  constant.DB_NAME = os.Getenv("DBDatabase")

  if constant.KEY == "" || constant.SECRET == "" {
    log.Panicln("Missing PaperKey or PaperSecret")
  }

  constant.AUTH_HEADERS = http.Header{
    "accept": {"application/json"},
    "APCA-API-KEY-ID": {constant.KEY},
    "APCA-API-SECRET-KEY": {constant.SECRET},
  }
}
