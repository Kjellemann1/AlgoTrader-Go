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

type mockAccountConn struct {
  *websocket.Conn
}

func (c *mockAccountConn) write(data string) {
  msg := []byte(data)
  _ = c.WriteMessage(1, msg)
}

func (c *mockAccountConn) read() string {
  _, msg, _ := c.ReadMessage()
  return string(msg)
}

func mockServerAccount(urlChan chan string, msgChan chan string, rootWg *sync.WaitGroup, signalChan chan int8) {
  defer rootWg.Done()
  var wg sync.WaitGroup
  wg.Add(3)  // wg = 3 since wg.Done() should be called 3 times due to reconnect
  iter := 0
  upgrader := websocket.Upgrader{}
  server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    ws, _ := upgrader.Upgrade(w, r, nil)
    conn := mockAccountConn{ws}
    defer conn.Close()
    if iter == 1 {
      msgChan <- conn.read()
      conn.write(`{"stream":"authorization,","data":{"status":"unathorized","action":"authenticate"}}`)
    } else if iter == 2 {
      msgChan <- conn.read()
      conn.write(`{"stream":"authorization","data":{"status":"unauthorized","action":"listen"}}`)
      conn.write(`{"stream":"authorization","data":{"status":"authorized","action":"authenticate"}}`)
      msgChan <- conn.read()
      conn.write(`{"stream":"listening","data":{"streams":["trade_updates"]}}`)
      signalChan <- 1
    } else {
      msgChan <- conn.read()
      conn.write(`{"stream":"authorization","data":{"status":"authorized","action":"authenticate"}}`)
      msgChan <- conn.read()
      conn.write(`{"stream":"listening","data":{"streams":["trade_updates"]}}`)
      signalChan <- 1
    }
    iter++
    wg.Done()
  }))
  defer server.Close()
  urlChan <- "ws" + strings.TrimPrefix(server.URL, "http")
  wg.Wait()
}

func TestAccountReconnect(t *testing.T) {
  assets := make(map[string]map[string]*Asset)
  assets["stock"] = make(map[string]*Asset)
  db_chan := make(chan *Query, 1)

  t.Run("pingPong error", func(t *testing.T) {
    urlChan := make(chan string)
    defer close(urlChan)
    signalChan := make(chan int8)
    defer close(signalChan)
    msgChan := make(chan string, 10)

    rootCtx, rootCancel := context.WithCancel(context.Background())

    var rootWg sync.WaitGroup
    rootWg.Add(1)

    go mockServerAccount(urlChan, msgChan, &rootWg, signalChan)


    a := NewAccount(assets, <-urlChan, db_chan)
    a.pingPong = func(ctx context.Context, wg *sync.WaitGroup, err_chan chan error) {
      defer wg.Done()
      <-signalChan
      err_chan <- errors.New("mockPingPong")
    }

    var wg sync.WaitGroup
    wg.Add(1)
    assert.Panics(t, func() { a.start(&wg, rootCtx, 0) }, "Expected panic after server close")
    rootCancel()

    rootWg.Wait()
    close(msgChan)

    subMsgCount := 0
    for range msgChan {
      subMsgCount++
    }

    assert.Equal(t, 5, subMsgCount)
  })

  t.Run("listen error", func(t *testing.T) {
    urlChan := make(chan string)
    defer close(urlChan)
    signalChan := make(chan int8)
    defer close(signalChan)
    msgChan := make(chan string, 10)

    rootCtx, rootCancel := context.WithCancel(context.Background())

    var rootWg sync.WaitGroup
    rootWg.Add(1)

    go mockServerAccount(urlChan, msgChan, &rootWg, signalChan)

    a := NewAccount(assets, <-urlChan, db_chan)
    a.pingPong = func(ctx context.Context, wg *sync.WaitGroup, err_chan chan error) {
      defer wg.Done()
      <-signalChan
      err_chan <- errors.New("mockPingPong")
    }

    var wg sync.WaitGroup
    wg.Add(1)
    assert.Panics(t, func() { a.start(&wg, rootCtx, 0) }, "Expected panic after server close")
    rootCancel()

    rootWg.Wait()
    close(msgChan)

    subMsgCount := 0
    for range msgChan {
      subMsgCount++
    }

    assert.Equal(t, 5, subMsgCount)
  })
}
