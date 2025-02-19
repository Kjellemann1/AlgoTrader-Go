package constant

import (
  "time"
  "net/http"
)

const (
  WINDOW_SIZE int = 500
  ORDER_AMOUNT_USD float64 = 500
  ENDPOINT = "https://paper-api.alpaca.markets/v2"
  WSS_STOCK = "wss://stream.data.alpaca.markets/v2/iex"
  WSS_CRYPTO = "wss://stream.data.alpaca.markets/v1beta3/crypto/us"
  HIST_DAYS = 1
  HIST_LIMIT = 10000
  HTTP_TIMEOUT_SEC = 5 * time.Second
  MAX_RECEIVED_TIME_DIFF_MS = 100 * time.Millisecond
  MAX_TRIGGER_TIME_DIFF_MS = 100 * time.Millisecond
)

var (
  AUTH_HEADERS http.Header
  KEY string
  SECRET string
  PUSH_TOKEN string
  PUSH_USER string
  DB_USER string
  DB_PASSWORD string
  DB_NAME string
  DB_HOST string
  DB_PORT string
)

var STOCK_LIST = []string{
  // DID YOU FIX FILL MISSING MINUTES YET?
  // "AAPL",
  // "MSFT",
  // "NVDA",
  // "GOOGL",
  // "AMZN",
  // "META",
  // "BRK.B",
  // "LLY",
  // "TSM",
  // "AVGO",
  // "TSLA",
  // "NVO",
  // "JPM",
  // "WMT",
  // "V",
  // "XOM",
  // "UNH",
  // "ASML",
  // "MA",
  // "ORCL",
  // "PG",
  // "COST",
  // "JNJ",
  // "HD",
  // "BAC",
  // "MRK",
  // "ABBV",
  // "AMD",
  // "CVX",
  // "NFLX",
}

var CRYPTO_LIST = []string{
  "BTC/USD",
  "ETH/USD",
  "USDT/USD",
  "SOL/USD",
  "USDC/USD",
  "DOGE/USD",
  "LINK/USD",
  "AVAX/USD",
  "LTC/USD",
  "SHIB/USD",
  "DOT/USD",
  "BCH/USD",
  "UNI/USD",
  "AAVE/USD",
  "YFI/USD",
  "MKR/USD",
  "GRT/USD",
  "XTZ/USD",
  "BAT/USD",
  "SUSHI/USD",
  "CRV/USD",
}
