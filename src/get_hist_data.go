
package src

import (
  "time"
  "fmt"
  "strings"
  "net/http"
  "log"
  "io"
  "github.com/valyala/fastjson"
  "github.com/Kjellemann1/AlgoTrader-Go/src/constant"
)


func urlHistBars(asset_class string, page_token string) string {
  t := time.Now().UTC().AddDate(0, 0, -constant.HIST_DAYS).Format("2006-01-02T15:04:05Z")
  // fmt.Println(strings.Replace(strings.Join(constant.CRYPTO_LIST, "%2C"), "/", "%2F", len(constant.CRYPTO_LIST)))
  // panic("")
  var url string
  switch asset_class {
    case "stock":
      url = fmt.Sprintf(
        "https://data.alpaca.markets/v2/stocks/bars?symbols=%s" +
        "&timeframe=1Min&start=%s&limit=%d&adjustment=all&feed=iex&",
        strings.Join(constant.STOCK_LIST, "%2C"), t, constant.HIST_LIMIT,
      )
    case "crypto":
      url = fmt.Sprintf(
        "https://data.alpaca.markets/v1beta3/crypto/us/bars?symbols=%s" +
        "&timeframe=1Min&start=%s&limit=%d&",
        strings.Replace(strings.Join(constant.CRYPTO_LIST, "%2C"), "/", "%2F", len(constant.CRYPTO_LIST)), 
        t, constant.HIST_LIMIT,
      )
  }
  if page_token != "start" {
    url += fmt.Sprintf("page_token=%s&", page_token)
  }
  url += "sort=asc"
  return url
}


func makeRequest(asset_class string, page_token string) *fastjson.Value {
  url := urlHistBars(asset_class, page_token)
  req, err := http.NewRequest("GET", url, nil)
  if err != nil {
    log.Fatalf(
      "[ ERROR ]\tFailed to create request in GetHistBars()\n" +
      "  -> Error: %s\n",
    err)
  }
  req.Header = constant.AUTH_HEADERS
  response, err := http.DefaultClient.Do(req)
  if err != nil {
    log.Fatalf(
      "[ ERROR ]\tFailed to send request in GetHistBars()\n" +
      "  -> Error: %s\n",
    err)
  }
  body, _ := io.ReadAll(response.Body)
  p := fastjson.Parser{}
  parsed, _ := p.ParseBytes(body)
  return parsed
}


func GetHistBars(assets map[string]*Asset, asset_class string) {
  var arr []*fastjson.Value
  page_token := "start"
  var x int = 0
  for page_token != "" {
    parsed := makeRequest(asset_class, page_token)
    arr = append(arr, parsed.Get("bars"))
    page_token = string(parsed.GetStringBytes("next_page_token"))
    x++
  }
  for _, bars := range arr {
    if bars == nil {
      fmt.Println("Bars is nil")
      continue
    }
    obj, err := bars.Object()
    if err != nil {
      log.Fatalf(
        "[ ERROR ]\tFailed to get object in GetHistBars()\n" +
        "  -> Error: %s\n",
      err)
    }
    obj.Visit(func(symbol []byte, value *fastjson.Value) {
      x := value.GetArray()
      for _, bar := range x {
        t, _ := time.Parse("2006-01-02T15:04:05Z", string(bar.GetStringBytes("t")))
        assets[string(symbol)].UpdateWindowOnBar(
          bar.GetFloat64("o"),
          bar.GetFloat64("h"),
          bar.GetFloat64("l"),
          bar.GetFloat64("c"),
          t,
          time.Now().UTC(),
          time.Now().UTC(),
        )
      }
    })
  }
}

// TODO: Add check that none of the prices are zero
