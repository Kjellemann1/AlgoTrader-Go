package main

import (
  "testing"
  "net/http"
  "net/http/httptest"
  "strings"
  "sync"
  "context"
  "github.com/gorilla/websocket"
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

func TestMarket(t *testing.T) {
  upgrader := websocket.Upgrader{}
  server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    ws, _ := upgrader.Upgrade(w, r, nil)
    conn := connection{ws}
    defer conn.Close()
    conn.write(`[{"T":"success","msg":"connected"}]`)
    conn.read()
    conn.write(`[{"T":"success","msg":"authenticated"}]`)
    conn.read()
    conn.write(`[{"T":"success","msg":"subscribed"}]`)
  }))
  defer server.Close()

  assets := make(map[string]*Asset)
  assets["FOO"] = newAsset("stock", "FOO")
  assets["BAR"] = newAsset("stock", "BAR")

  var wg sync.WaitGroup
  wg.Add(1)

  rootCtx, rootCancel := context.WithCancel(context.Background())

  ws_url := "ws" + strings.TrimPrefix(server.URL, "http")
  m := NewMarket("stock", ws_url, assets)
  go m.start(&wg, rootCtx)

  rootCancel()
}
