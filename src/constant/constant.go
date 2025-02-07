
package constant

import (
  "net/http"
  "time"
)


const WINDOW_SIZE int = 500

const ORDER_AMOUNT_USD float64 = 500

const ENDPOINT = "https://paper-api.alpaca.markets/v2"

var AUTH_HEADERS http.Header
var KEY string
var SECRET string

var PUSH_TOKEN string
var PUSH_USER string

var DB_USER string
var DB_PASSWORD string
var DB_NAME string
var DB_HOST string
var DB_PORT string


const HIST_DAYS = 3
const HIST_LIMIT = 10000


const WSS_STOCK = "wss://stream.data.alpaca.markets/v2/iex"
const WSS_CRYPTO = "wss://stream.data.alpaca.markets/v1beta3/crypto/us"

const HTTP_TIMEOUT_SEC = 5 * time.Second

const MAX_RECEIVED_TIME_DIFF_MS = 100 * time.Millisecond
const MAX_TRIGGER_TIME_DIFF_MS = 100 * time.Millisecond


var STOCK_LIST = []string{
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
  "LTC/USD",
  "ETH/USD",
  "SHIB/USD",
  "AAVE/USD",
  "AVAX/USD",
  "BAT/USD",
  "BCH/USD",
  "CRV/USD",
  "DOGE/USD",
  "DOT/USD",
  "GRT/USD",
  "LINK/USD",
  "MKR/USD",
  "SOL/USD",
  "SUSHI/USD",
  "UNI/USD",
  "USDC/USD",
  "USDT/USD",
  "XTZ/USD",
  "YFI/USD",
}
