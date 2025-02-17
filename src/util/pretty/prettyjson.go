package pretty

import (
  "encoding/json"
  "bytes"
  "log"
  "github.com/valyala/fastjson"
)

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
