
package push

import (
  "fmt"
  "bytes"
  "net/http"

  "github.com/Kjellemann1/AlgoTrader-Go/src/constant"
)


func push(message string, title string, prio int) {
  url := "https://api.pushover.net/1/messages.json"
  devices := []string{"Zenfone10"}
  for i := 0; i < len(devices); i++ {
    var data string
    if prio == 2 {
      data = fmt.Sprintf(`{
        "token": "%s",
        "user": "%s",
        "device": "%s",
        "title": "%s",
        "message": "%s",
        "priority": %d,
        "expire": 3600,
        "retry": 60
      }`, constant.PUSH_TOKEN, constant.PUSH_USER, devices[i], title, message, prio)
    } else {
      data = fmt.Sprintf(`{
        "token": "%s",
        "user": "%s",
        "device": "%s",
        "title": "%s",
        "message": "%s",
        "priority": %d
      }`, constant.PUSH_TOKEN, constant.PUSH_USER, devices[i], title, message, prio)
    }
    response, err := http.Post(url, "application/json", bytes.NewBuffer([]byte(data)))
    if err != nil {
      fmt.Println("Error making POST request:", err)
      continue
    }
    defer response.Body.Close()
    if response.StatusCode != http.StatusOK {
      fmt.Println("Failed to send push notification. Status:", response.Status)
    }
  }
}


func Info(message string) {
  push(message, "UPDATE", -1)
}


func Message(message string) {
  push(message, "MESSAGE", 0)
}


func Warning(message string, err error) {
  if err != nil {
    x := fmt.Sprintf("%s\n  -> Error: %s", message, err.Error())
    push(x, "WARNING", 1)
  } else {
    push(message, "WARNING", 1)
  }
}


func Error(message string, err error) {
  if err != nil {
    x := fmt.Sprintf("%s\n  -> Error: %s", message, err.Error())
    push(x, "ERROR", 2)
  } else {
    push(message, "ERROR", 2)
  }
}
