package constant

import (
  "time"
  "net/http"
)

const (
  ENDPOINT = "https://paper-api.alpaca.markets/v2"
  WSS_STOCK = "wss://stream.data.alpaca.markets/v2/iex"
  WSS_CRYPTO = "wss://stream.data.alpaca.markets/v1beta3/crypto/us"
  WSS_ACCOUNT = "wss://paper-api.alpaca.markets/stream"
  WINDOW_SIZE int = 500
  NOTIONAL_USD float64 = 50
  HIST_DAYS = 1
  HIST_LIMIT = 10000
  HTTP_TIMEOUT_SEC = 5 * time.Second
  MAX_RECEIVED_TIME_DIFF_MS = 100 * time.Millisecond
  MAX_TRIGGER_TIME_DIFF_MS = 100 * time.Millisecond
  READ_DEADLINE_SEC = 20 * time.Second
  PING_INTERVAL_SEC = 10 * time.Second
  RATE_LIMIT_SLEEP_SEC = 30 * time.Second
  REQUEST_RETRIES = 4
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

var CRYPTO_SYMBOLS = []string{
  "BTC/USD",
  "ETH/USD",
  "USDT/USD",
  "SOL/USD",
  "USDC/USD",
  "DOGE/USD",
  "LINK/USD",
  "AVAX/USD",
  // Currencies below this line are quite illiquid
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

var STOCK_SYMBOLS = []string{
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
