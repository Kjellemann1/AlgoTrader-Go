package main

import (
  "errors"
  "testing"
  "net/http"
  "net/http/httptest"
  "strings"
  "sync"
  "context"
  "github.com/gorilla/websocket"
  "github.com/stretchr/testify/assert"
)

type connection struct {
  *websocket.Conn
}

func (c *connection) write(data string) {
  msg := []byte(data)
  _ = c.WriteMessage(1, msg)
}

func (c *connection) read() string {
  _, msg, _ := c.ReadMessage()
  return string(msg)
}

func mockServer (urlChan chan string, msgChan chan string, rootWg *sync.WaitGroup, signalChan chan int8) {
  defer rootWg.Done()
  var wg sync.WaitGroup
  wg.Add(3)  // wg = 3 since wg.Done() should be called 3 times due to reconnect
  iter := 0
  upgrader := websocket.Upgrader{}
  server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    ws, _ := upgrader.Upgrade(w, r, nil)
    conn := connection{ws}
    defer conn.Close()
    if iter == 1 {
      conn.write(`[{"T":"error","msg":"mockPingPong"}]`)
    } else {
      conn.write(`[{"T":"success","msg":"connected"}]`)
      conn.write(`[{"T":"success","msg":"authenticated"}]`)
      msgChan <- conn.read()
      conn.write(`[{"T":"subscription","trades":["FOO","BAR"],"bars":["FOO","BAR"]}]`)
      signalChan <- 1
    }
    iter++
    wg.Done()
  }))
  defer server.Close()
  urlChan <- "ws" + strings.TrimPrefix(server.URL, "http")
  wg.Wait()
}

func TestMarketReconnect(t *testing.T) {
  assets := make(map[string]*Asset)
  assets["FOO"] = newAsset("stock", "FOO")
  assets["BAR"] = newAsset("stock", "BAR")

  t.Run("pingPong error", func(t *testing.T) {
    urlChan := make(chan string)
    msgchan := make(chan string)
    signalChan := make(chan int8)
    defer close(msgchan)
    defer close(urlChan)

    rootCtx, rootCancel := context.WithCancel(context.Background())

    var rootWg sync.WaitGroup
    rootWg.Add(1)

    var xWg sync.WaitGroup
    xWg.Add(1)

    go mockServer(urlChan, msgchan, &rootWg, signalChan)

    m := NewMarket("stock", <-urlChan, assets)
    m.pingPong = func(ctx context.Context, wg *sync.WaitGroup, err_chan chan error) {
      defer wg.Done()
      <-signalChan
      err_chan <- errors.New("mockPingPong")
    }

    subMsgCount := 0
    rootWg.Add(1)
    go func() {
      defer rootWg.Done()
      for range 2 {
        <-msgchan
        subMsgCount++
      }
    }()

    assert.Panics(t, func() { m.start(&xWg, rootCtx, 0) })  // Should panic after server is closed

    rootWg.Wait()

    rootCancel()

    assert.Equal(t, 2, subMsgCount)
  })

  t.Run("listen error", func(t *testing.T) {
    urlChan := make(chan string)
    msgchan := make(chan string, 10)
    signalChan := make(chan int8)
    defer close(msgchan)
    defer close(urlChan)

    rootCtx, rootCancel := context.WithCancel(context.Background())

    var rootWg sync.WaitGroup
    rootWg.Add(1)

    var xWg sync.WaitGroup
    xWg.Add(1)

    go mockServer(urlChan, msgchan, &rootWg, signalChan)

    m := NewMarket("stock", <-urlChan, assets)
    m.listen = func(ctx context.Context, wg *sync.WaitGroup, err_chan chan error) {
      defer wg.Done()
      <-signalChan
      err_chan <- errors.New("mockListen")
    }

    subMsgCount := 0
    rootWg.Add(1)
    go func() {
      defer rootWg.Done()
      for range 2 {
        <-msgchan
        subMsgCount++
      }
    }()

    assert.Panics(t, func() { m.start(&xWg, rootCtx, 0) })  // Should panic after server is closed

    rootWg.Wait()
    rootCancel()

    assert.Equal(t, 2, subMsgCount)
  })
}
