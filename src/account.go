
package src

import (
  "fmt"

  "github.com/gorilla/websocket"
)


type Account struct {
  conn *websocket.Conn

}

func (m *Account) connect(url string) {
  conn, response, err := websocket.DefaultDialer.Dial(url, AUTH_HEADERS)
  if err != nil {
    if response != nil {
      fmt.Println("Response: ", response)
    }
    panic(err)
  }
  m.conn = conn
}

func (m *Account) listen() {
  for {
    _, message, err := m.conn.ReadMessage()
    if err != nil {
      fmt.Println("Error reading message: ", err)
      panic(err)
    }
    // m.parse(string(message))
    fmt.Println("Message: ", string(message))
  }
}
