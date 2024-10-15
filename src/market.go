
package src

import (
  "fmt"

  "github.com/valyala/fastjson"
  "github.com/gorilla/websocket"
  // "github.com/shopspring/decimal" // https://pkg.go.dev/github.com/shopspring/decimal#section-readme

  "github.com/Kjellemann1/AlgoTrader-Go/src/util"
)


type Bar struct {
  O float64
  H float64
  L float64
  C float64
}


type Trade struct {
  Price float64
  Time string
}


type Market struct {
  asset_class string
  query_chan chan string
  position_monitor_chan chan map[string]string
  conn *websocket.Conn
  url string
  parser fastjson.Parser
}


func (m *Market) connect() {
  // Connect to the websocket
  conn, _, err := websocket.DefaultDialer.Dial(m.url, AUTH_HEADERS)
  if err != nil {
    panic(err)
  }
  m.conn = conn

  // Receive connection message
  _, message, err := m.conn.ReadMessage()
  if err != nil {
    panic(err)
  }
  if string(message) == `[{"T":"success","msg":"connected"}]` {
    fmt.Println("Connected")
  } else {
    fmt.Println(string(message))
    panic("Connecting failed")
  }
  // Receive authentication message
  _, message, err = m.conn.ReadMessage()
  if err != nil {
    panic(err)
  }
  if string(message) == `[{"T":"success","msg":"authenticated"}]` {
    fmt.Println("Authenticated")
  } else {
    fmt.Println(string(message))
    panic("Authenticating failed")
  }
  // Subscbribe to symbols
  message = []byte(`{"action":"subscribe","trades":["FAKEPACA"], "bars":["FAKEPACA"]}`)
  if err := m.conn.WriteMessage(websocket.TextMessage, message); err != nil {
    panic(err)
  }
}


func (m *Market) messageHandler(message []byte) {
  arr, err := m.parser.ParseBytes(message)
  util.AssertNotNil(err, "Error parsing json")
  util.AssertTrue(arr.Type() == fastjson.TypeArray, "Message is not an array")
  for _, element := range arr.GetArray() {
    message_type := string(element.GetStringBytes("T"))
    switch message_type {
    case "b":
      bar := m.BarParser(element)
      util.PrettyPrint(bar)
    case "t":
      trade := m.TradeParser(element)
      util.PrettyPrint(trade)
    }
  }
}


func (m *Market) BarParser(element *fastjson.Value) Bar {
  bar := Bar{
    O: element.GetFloat64("o"),
    H: element.GetFloat64("h"),
    L: element.GetFloat64("l"),
    C: element.GetFloat64("c"),
  }
  return bar
}


func (m *Market) TradeParser(element *fastjson.Value) Trade {
  trade := Trade{
    Price: element.GetFloat64("p"),
    Time:  string(element.GetStringBytes("t")),
  }
  return trade
}


func (m *Market) listen() {
  for {
    _, message, err := m.conn.ReadMessage()
    util.AssertNotNil(err, "Error reading message")
    m.messageHandler(message)
  }
}


func (m *Market) Init() {
  m.connect()
  m.listen()
}
