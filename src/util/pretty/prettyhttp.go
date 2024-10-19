
package pretty

import (
  "net/http"
  "fmt"
  "io"
)

func ResponseToString(res *http.Response) string {
  // Hent statuslinje
  result := fmt.Sprintf("%s %s\n", res.Proto, res.Status)
  
  // Legg til headers
  for name, headers := range res.Header {
    for _, header := range headers {
      result += fmt.Sprintf("%s: %s\n", name, header)
    }
  }

  // Legg til body hvis tilgjengelig
  if res.Body != nil {
    bodyBytes, err := io.ReadAll(res.Body)
    if err == nil {
      result += fmt.Sprintf("\n%s", string(bodyBytes))
    }
    // Husk å gjenopprette Body etter lesing
    res.Body = io.NopCloser(io.NopCloser(nil))
  }

  return result
}


func RequestToString(req *http.Request) string {
  // Hent metode, URL, og headers
  result := fmt.Sprintf("%s %s %s\n", req.Method, req.URL, req.Proto)
  
  // Legg til headers
  for name, headers := range req.Header {
    for _, header := range headers {
      result += fmt.Sprintf("%s: %s\n", name, header)
    }
  }

  // Legg til body hvis tilgjengelig
  if req.Body != nil {
    bodyBytes, err := io.ReadAll(req.Body)
    if err == nil {
      result += fmt.Sprintf("\n%s", string(bodyBytes))
    }
    // Husk å gjenopprette Body etter lesing
    req.Body = io.NopCloser(io.NopCloser(nil))
  }

  return result
}

