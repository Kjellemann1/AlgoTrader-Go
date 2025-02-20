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
  name := "/var/lib/mysql-files/algologs/" + time.Now().In(time.UTC).Format(time.DateTime) + ".log"
  logfile, err := os.OpenFile(name, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
  if err != nil {
    panic(err)
  }
  multiWriter := io.MultiWriter(os.Stdout, logfile)
  log.SetOutput(multiWriter)
}

func init() {
  err := godotenv.Load()
  if err != nil {
    panic(err)
  }

  constant.PUSH_TOKEN = os.Getenv("PushoverToken")
  constant.PUSH_USER = os.Getenv("PushoverUser")

  constant.KEY = os.Getenv("PaperKey")
  constant.SECRET = os.Getenv("PaperSecret")

  constant.DB_USER = os.Getenv("DBUser")
  constant.DB_PASSWORD = os.Getenv("DBPassword")
  constant.DB_NAME = os.Getenv("DBDatabase")
  constant.DB_HOST = os.Getenv("DBHost")
  constant.DB_PORT = os.Getenv("DBPort")

  if constant.KEY == "" || constant.SECRET == "" {
    panic("Missing PaperKey or PaperSecret")
  }

  constant.AUTH_HEADERS = http.Header{
    "accept": {"application/json"},
    "APCA-API-KEY-ID": {constant.KEY},
    "APCA-API-SECRET-KEY": {constant.SECRET},
  }
}
