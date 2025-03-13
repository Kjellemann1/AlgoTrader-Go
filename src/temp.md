
# Reconnect not triggered

I got an error when running my program in pingPongFunc. The error was : "write tcp 'ip1'->'ip2: write: broken pipe". As you can see from the function, this then sends error to the err_chan where it is picked up in start where cancel() is called in order to close the go routine running listenFucn(). However, this did not shutdown that go routine, and the program then just kept running and I was not receiving updates from the websocket connection.

What I need to happen is that when I get an error in either a.listen or a.pingPong, both of the go-routines need to be shutdown and then a reconnect triggered. 

I have tried to write tests to make sure this functionality works as intended, however my tests pass, but I still got the error I described above.

Please help me, I have been struggling to get this functionality right for some time now.

# Selected code from account.go

type Account struct {
  conn    *websocket.Conn
  parser  fastjson.Parser
  db_chan chan *Query
  assets  map[string]map[string]*Asset
  url     string

  listen func(context.Context, *sync.WaitGroup, chan error)
  pingPong func(context.Context, *sync.WaitGroup, chan error)
}

func NewAccount(assets map[string]map[string]*Asset, url string, db_chan chan *Query) *Account {
  a := &Account{
    parser: fastjson.Parser{},
    assets: assets,
    db_chan: db_chan,
    url: url,
  }
  a.pingPong = a.pingPongFunc
  a.listen = a.listenFunc
  return a
}

func (a *Account) pingPongFunc(ctx context.Context, connWg *sync.WaitGroup, err_chan chan error) {
  defer connWg.Done()

  if err := a.conn.SetReadDeadline(time.Now().Add(constant.READ_DEADLINE_SEC)); err != nil {
    util.Warning(err)
  }

  a.conn.SetPongHandler(func(string) error {
    err := a.conn.SetReadDeadline(time.Now().Add(constant.READ_DEADLINE_SEC))
    if err != nil {
      util.Warning(err)
    }
    return nil
  })

  ticker := time.NewTicker(constant.PING_INTERVAL_SEC)
  defer ticker.Stop()
  util.Ok("PingPong initiated for account websocket")

  for {
    select {
    case <-ctx.Done():
      return
    case <-ticker.C:
      if err := a.conn.WriteControl(
        websocket.PingMessage, []byte("ping"), 
        time.Now().Add(5 * time.Second)); err != nil {
        util.Error(err)
        err_chan <-err
        return
      }
    }
  }
}

func (a *Account) listenFunc(ctx context.Context, connWg *sync.WaitGroup, err_chan chan error) {
  defer connWg.Done()
  for {
    _, message, err := a.conn.ReadMessage()
    if err != nil {
      select {
      case <-ctx.Done():
        return
      default:
        util.Error(err)
        err_chan <-err
        return
      }
    }
    _ = a.messageHandler(message)
  }
}

func (a *Account) start(wg *sync.WaitGroup, ctx context.Context, backoff_sec_min float64) {
  defer wg.Done()

  backoff_sec := backoff_sec_min
  retries := 0

  for {
    if err := a.connect() ; err != nil {
      if retries < 5 {
        util.Error(err, "Retries", retries)
        util.Backoff(&backoff_sec)
        retries++
        continue
      } else {
        NNP.NoNewPositionsTrue("")
        util.Error(err, "Max retries reached", retries, "CLOSING ALL POSITIONS AND SHUTTING DOWN", "...")
        request.CloseAllPositions(2, 0)
        log.Panicln("SHUTTING DOWN")
      }
    }

    backoff_sec = backoff_sec_min
    retries = 0

    err_chan := make(chan error, 1)
    childCtx, cancel := context.WithCancel(ctx)

    a.checkPending()

    var connWg sync.WaitGroup

    connWg.Add(1)
    go a.listen(childCtx, &connWg, err_chan)

    connWg.Add(1)
    go a.pingPong(childCtx, &connWg, err_chan)

    select {
    case <-ctx.Done():
      cancel()
      a.conn.Close()
      connWg.Wait()
      return
    case <-err_chan:
      cancel()
      a.conn.Close()
      connWg.Wait()
    }
  }
}


# Tests I have written for this retry logic in particular

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
