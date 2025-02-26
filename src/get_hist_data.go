package main

import (
  "time"
  "fmt"
  "strings"
  "log"
  "github.com/valyala/fastjson"
  "github.com/Kjellemann1/AlgoTrader-Go/constant"
  "github.com/Kjellemann1/AlgoTrader-Go/request"
)

func urlHistBars(asset_class string, page_token string) string {
  t := time.Now().UTC().AddDate(0, 0, -constant.HIST_DAYS).Format("2006-01-02T15:04:05Z")
  var url string
  switch asset_class {
    case "stock":
      url = fmt.Sprintf(
        "https://data.alpaca.markets/v2/stocks/bars?symbols=%s" +
        "&timeframe=1Min&start=%s&limit=%d&adjustment=all&feed=iex&",
        strings.Join(constant.STOCK_SYMBOLS, "%2C"), t, constant.HIST_LIMIT,
      )
    case "crypto":
      url = fmt.Sprintf(
        "https://data.alpaca.markets/v1beta3/crypto/us/bars?symbols=%s" +
        "&timeframe=1Min&start=%s&limit=%d&",
        strings.Replace(strings.Join(constant.CRYPTO_SYMBOLS, "%2C"), "/", "%2F", len(constant.CRYPTO_SYMBOLS)), 
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
  body, err := request.GetReq(url)
  if err != nil {
    log.Fatal(err.Error())
  }
  p := fastjson.Parser{}
  parsed, err := p.ParseBytes(body)
  if err != nil {
    log.Fatal(err.Error())
  }
  return parsed
}

func fillRollingWindows (assets map[string]map[string]*Asset) {
  for k, v := range assets {
    getHistBars(v, k)
  }
}

func getHistBars(assets map[string]*Asset, asset_class string) {
  var arr []*fastjson.Value
  page_token := "start"
  for page_token != "" {
    parsed := makeRequest(asset_class, page_token)
    arr = append(arr, parsed.Get("bars"))
    page_token = string(parsed.GetStringBytes("next_page_token"))
  }
  temp_time := time.Now().UTC()
  for _, bars := range arr {
    if bars == nil {
      log.Println("[ WARNING ]\tBars is nil")
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
        assets[string(symbol)].updateWindowOnBar(
          bar.GetFloat64("o"),
          bar.GetFloat64("h"),
          bar.GetFloat64("l"),
          bar.GetFloat64("c"),
          t,
          temp_time,
        )
      }
    })
  }

  checkForZeroVals(assets)
}

func checkForZeroVals(assets map[string]*Asset) {
  // API returns zero in place of missing data
  for _, asset := range assets {
    for i := range constant.WINDOW_SIZE {
      if asset.O[i] == 0 || asset.H[i] == 0 || asset.L[i] == 0 || asset.C[i] == 0 {
        log.Println("[ ERROR ]\tZero values in window")
      }
    }
  }
}
