package push

import (
  "fmt"
  "log"
  "bytes"
  "net/http"

  "github.com/Kjellemann1/AlgoTrader-Go/constant"
)

var httpClient = &http.Client{}

func push(message string, title string, prio int) {
  url := "https://api.pushover.net/1/messages.json"
  var payload string
  if prio == 2 {
    payload = `{` +
      `"token": "` + constant.PUSH_TOKEN + `",` +
      `"user": "` + constant.PUSH_USER + `",` +
      `"title": "` + title + `",` +
      `"message": "` + message + `",` +
      `"priority": ` + fmt.Sprint(prio) + `,` +
      `"expire": 3600,` +
      `"retry": 60` +
    `}`
  } else {
    payload = `{` +
      `"token": "` + constant.PUSH_TOKEN + `",` +
      `"user": "` + constant.PUSH_USER + `",` +
      `"title": "` + title + `",` +
      `"message": "` + message + `",` +
      `"priority": ` + fmt.Sprint(prio) +
    `}`
  }
  response, err := httpClient.Post(url, "application/json", bytes.NewBuffer([]byte(payload)))
  if err != nil {
    log.Printf(
      "[ WARNING ]\tError making POST request\n  -> Error: %s\n  -> Response status: %s\n Payload: %s", err, response.Status, payload)
    return
  }
  defer response.Body.Close()
  if response.StatusCode != http.StatusOK {
    log.Printf(
      "[ WARNING ]\tFailed to send push notification\n  -> Response status: %s\n  -> Payload: %s\n", response.Status, payload)
  }
}

func Info(message string) {
  push(message, "UPDATE", -1)
}

func Message(message string) {  // Change to 0
  push(message, "MESSAGE", 0)
}

func Warning(message string) {
  push(message, "WARNING", 0)  // Change to 1
}

func Error(message string) {
  push(message, "ERROR", 0)  // Change to 2
}

// Everything below this line is for testing purposes
func DisablePush() {
  httpClient = &http.Client{
    Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
      return &http.Response{ StatusCode: 200 }, nil
    }),
  }
}

type roundTripFunc func(req *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
  return f(req)
}
