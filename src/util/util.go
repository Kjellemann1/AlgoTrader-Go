package util

import (
  "time"
  "fmt"
  "encoding/json"
  "bytes"
  "log"
  "github.com/valyala/fastjson"
  "runtime"
  "errors"
  "path/filepath"
  "github.com/Kjellemann1/AlgoTrader-Go/push"
)

func BackoffWithMax(backoff_sec *float64, backoff_max_sec float64) {
  time.Sleep(time.Duration(*backoff_sec) * time.Second)
  *backoff_sec = min(*backoff_sec * 2, backoff_max_sec)
}

func Backoff(backoff_sec *float64) {
  time.Sleep(time.Duration(*backoff_sec) * time.Second)
  *backoff_sec = *backoff_sec * 2
}

func PrintFormattedJSON(v *fastjson.Value) {
  jsonBytes := v.MarshalTo(nil)
  var buf bytes.Buffer
  err := json.Indent(&buf, jsonBytes, "", "  ")
  if err != nil {
    log.Printf("Feil ved formatering av JSON: %v\n", err)
    return
  }
  log.Println(buf.String())
}

func AddWhitespace(s string, n int) string {
  for len(s) < n {
    s += " "
  }
  return s
}

func Info(message string, details ...any) {
  logMsg := "[ INFO ]\t" + message
  for i := 0; i < len(details); i += 2 {
    key, ok := details[i].(string)
    if !ok {
      logMsg += "\n  -> Invalid key (not a string): " + fmt.Sprint(details[i])
      continue
    }
    logMsg += "\n  -> " + key + ": " + fmt.Sprint(details[i+1])
  }
  log.Println(logMsg)
  push.Info(logMsg)
}

var Error = ErrorFunc
func ErrorFunc(err error, details ...any) {
  if err == nil {
    log.Println(errors.New("Called with nil error"))
    return
  }
  _, file, line, ok := runtime.Caller(1)
  if !ok {
    file = "unknown"
    line = -1
  } else {
    file = filepath.Base(file)
  }
  logMsg := "[ ERROR ]\t" + err.Error() + "\n  -> " + file + ":" + fmt.Sprint(line)
  for i := 0; i < len(details); i += 2 {
    key, ok := details[i].(string)
    if !ok {
      logMsg += "\n  -> Invalid key (not a string): " + fmt.Sprint(details[i])
      continue
    }
    logMsg += "\n  -> " + key + ": " + fmt.Sprint(details[i+1])
  }
  log.Println(logMsg)
  push.Error(logMsg)
}

func ErrorPanic(err error, details ...any) {
  if err == nil {
    log.Println(errors.New("Called with nil error"))
    return
  }
  _, file, line, ok := runtime.Caller(1)
  if !ok {
    file = "unknown"
    line = -1
  } else {
    file = filepath.Base(file)
  }
  logMsg := "[ ERROR ]\t" + err.Error() + "\n  -> " + file + ":" + fmt.Sprint(line)
  for i := 0; i < len(details); i += 2 {
    key, ok := details[i].(string)
    if !ok {
      logMsg += "\n  -> Invalid key (not a string): " + fmt.Sprint(details[i])
      continue
    }
    logMsg += "\n  -> " + key + ": " + fmt.Sprint(details[i+1])
  }
  log.Panicln(logMsg)
  push.Error(logMsg)
}

var Warning = WarningFunc
func WarningFunc(err error, details ...any) {
  if err == nil {
    log.Println(errors.New("Called with nil error"))
    return
  }
  _, file, line, ok := runtime.Caller(1)
  if !ok {
    file = "unknown"
    line = -1
  } else {
    file = filepath.Base(file)
  }
  logMsg := "[ WARNING ]\t" + err.Error() + "\n  -> " + file + ":" + fmt.Sprint(line)
  for i := 0; i < len(details); i += 2 {
    key, ok := details[i].(string)
    if !ok {
      logMsg += "\n  -> Invalid key (not a string): " + fmt.Sprint(details[i])
      continue
    }
    logMsg += "\n  -> " + key + ": " + fmt.Sprint(details[i+1])
  }
  log.Println(logMsg)
  push.Warning(logMsg)
}

// Caller 2
func Error2(err error, details ...any) {
  if err == nil {
    log.Println(errors.New("Called with nil error"))
    return
  }

  _, file, line, ok := runtime.Caller(2)
  if !ok {
    file = "unknown"
    line = -1
  } else {
    file = filepath.Base(file)
  }
  logMsg := "[ ERROR ]\t" + err.Error() + "\n  -> " + file + ":" + fmt.Sprint(line)
  for i := 0; i < len(details); i += 2 {
    key, ok := details[i].(string)
    if !ok {
      logMsg += "\n  -> Invalid key (not a string): " + fmt.Sprint(details[i])
      continue
    }
    logMsg += "\n  -> " + key + ": " + fmt.Sprint(details[i+1])
  }
  log.Println(logMsg)
  push.Error(logMsg)
}

func Warning2(err error, details ...any) {
  if err == nil {
    log.Println(errors.New("Called with nil error"))
    return
  }
  _, file, line, ok := runtime.Caller(2)
  if !ok {
    file = "unknown"
    line = -1
  } else {
    file = filepath.Base(file)
  }
  logMsg := "[ WARNING ]\t" + err.Error() + "\n  -> " + file + ":" + fmt.Sprint(line)
  for i := 0; i < len(details); i += 2 {
    key, ok := details[i].(string)
    if !ok {
      logMsg += "\n  -> Invalid key (not a string): " + fmt.Sprint(details[i])
      continue
    }
    logMsg += "\n  -> " + key + ": " + fmt.Sprint(details[i+1])
  }
  log.Println(logMsg)
  push.Warning(logMsg)
}
