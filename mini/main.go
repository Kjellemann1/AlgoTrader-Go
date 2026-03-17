package main

import (
  "io"
  "os"
  "fmt"
  "log"
  "time"
  "sync"
  "bytes"
  "net/http"
  "context"
  "github.com/gorilla/websocket"
  "github.com/joho/godotenv"
)

var (
  key       string
  secret    string
  pushtoken string
  pushuser  string
)

func init() {
  _ = godotenv.Load()

  
  name := "/root/Algo/minilogs/" + time.Now().In(time.UTC).Format(time.DateTime) + ".log"
  logfile, err := os.OpenFile(name, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
  if err != nil {
    log.Panicln(err)
  }
  multiWriter := io.MultiWriter(os.Stdout, logfile)
  log.SetOutput(multiWriter)

  key = os.Getenv("PaperKey")
  secret = os.Getenv("PaperSecret")
  pushtoken = os.Getenv("PushoverToken")
  pushuser = os.Getenv("PushoverUser")
}

type Account struct {
  conn *websocket.Conn
  err_chan chan int8
  url string
}

func (a *Account) connect() {
  a.conn, _, _ = websocket.DefaultDialer.Dial(a.url, nil)

  _ = a.conn.WriteMessage(websocket.TextMessage, 
    fmt.Appendf(make([]byte, 0), `{"action":"auth","key":"%s","secret":"%s"}`,
    key, secret),
  )

  _, message, _ := a.conn.ReadMessage()
  log.Println("Message:", string(message))

  _ = a.conn.WriteMessage(websocket.TextMessage,
    []byte(`{"action":"listen","data":{"streams":["trade_updates"]}}`),
  )

  _, message, _ = a.conn.ReadMessage()
  log.Println("Message:", string(message))
}

func (a *Account) setReadDeadline() {
  err := a.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
  if err != nil {
    push()
    log.Println("Error", err.Error())
  }
}

func (a *Account) setPongHandler() {
  a.conn.SetPongHandler(func(string) error {
    log.Println("Received pong")
    a.setReadDeadline()
    return nil
  })
}

func (a *Account) pinger(ctx context.Context, wg *sync.WaitGroup) {
  defer wg.Done()
  ticker := time.NewTicker(30 * time.Second)
  for {
    select {
    case <-ticker.C:
      log.Println("Sending ping")
      err := a.conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(5 * time.Second))
      if err != nil {
        push()
        log.Println("pinger error:", err.Error())
        select {
        case a.err_chan<-1:
        default:
        }
        return
      }
    case <-ctx.Done():
      log.Println("pinger ctx.Done")
      return
    }
  }
}

func (a *Account) listen(ctx context.Context, wg *sync.WaitGroup) {
  defer wg.Done()
  a.setPongHandler()
  for {
    log.Println("Listening")
    _, message, err := a.conn.ReadMessage()
    if err != nil {
      push()
      log.Println("listen error:", err.Error())
      select {
      case <-ctx.Done():
        log.Println("listen ctx.Done")
        return
      case a.err_chan<-1:
        log.Println("listen err_chan")
      default:
      }
    }
    log.Println("Message:", string(message))
  }
}

func (a *Account) start(rootCtx context.Context, rootWg *sync.WaitGroup) {
  defer rootWg.Done()
  for {
    log.Println("Connecting ...")
    a.connect()
    log.Println("Connected")

    a.setReadDeadline()

    var wg sync.WaitGroup
    ctx, cancel := context.WithCancel(rootCtx)
    wg.Add(2)
    go a.pinger(ctx, &wg)
    go a.listen(ctx, &wg)

    select {
    case <-ctx.Done():
      cancel()
      if err := a.conn.Close(); err != nil {
        log.Println("Error in start 1", err.Error())
      }
      wg.Wait()
      return
    case <-a.err_chan:
      log.Println("Error received")
      cancel()
      if err := a.conn.Close(); err != nil {
        log.Println("Error in start 2", err.Error())
      }
      wg.Wait()
    }
  }
}

func push() {
  url := "https://api.pushover.net/1/messages.json"
  payload := `{` +
    `"token": "` + pushtoken + `",` +
    `"user": "` + pushuser + `",` +
    `"title": "mini",` +
    `"message": "mini",` +
    `"priority": 0` +
  `}`
  response, err := http.DefaultClient.Post(url, "application/json", bytes.NewBuffer([]byte(payload)))
  if err != nil {
    log.Println("push error:", err.Error())
  }
  if response.StatusCode != http.StatusOK {
    log.Println("push response status:", response.Status)
  }
  defer response.Body.Close()
}

func main() {
  push()
  context, cancel := context.WithCancel(context.Background())
  var wg sync.WaitGroup

  wg.Add(1)
  c := Account{
    err_chan: make(chan int8),
    url: "wss://paper-api.alpaca.markets/stream",
  }
  go c.start(context, &wg)

  wg.Wait()
  cancel()

  log.Println("Exiting")
}
