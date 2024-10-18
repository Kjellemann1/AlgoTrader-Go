
// Printing and logging json in a more readable format

package prettyjson

import (
  "encoding/json"
  "fmt"
  "log"
)


func Print(i interface{}) {
  s, err := json.MarshalIndent(i, "", "\t")
  if err != nil {
    log.Println("Error marshalling json in print.PrettyPrint(): ", err)
    log.Println("  -> Raw json: " + string(s), nil)
  } else {
    fmt.Println(string(s))
  }
}


func Format(i interface{}) string {
  s, err := json.MarshalIndent(i, "", "\t")
  if err != nil {
    log.Printf("Error marshalling json in print.PrettyFormat(): %s\n  -> json: %s", err, string(s))
    return ""
  } else {
    return string(s)
  }
}
