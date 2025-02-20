
package handlelog

import (
  "fmt"
  "log"
  "runtime"
  "errors"
  "path/filepath"
  "github.com/Kjellemann1/AlgoTrader-Go/util/push"
)


func Info(message string, details ...interface{}) {
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


func Error(err error, details ...interface{}) {
  if err == nil {
    err = errors.New("Called with nil error")
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

func ErrorPanic(err error, details ...interface{}) {
  if err == nil {
    err = errors.New("Called with nil error")
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


func Warning(err error, details ...interface{}) {
  if err == nil {
    err = errors.New("Called with nil error")
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

func Error2(err error, details ...interface{}) {
  if err == nil {
    err = errors.New("Called with nil error")
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


func Warning2(err error, details ...interface{}) {
  if err == nil {
    err = errors.New("Called with nil error")
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
