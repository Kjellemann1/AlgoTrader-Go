
package constant

import "net/http"


const WINDOW_SIZE int = 500

const CHANNEL_BUFFER_SIZE int = 50
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
  // "SHIB/USD",
  // "AAVE/USD",
  // "AVAX/USD",
  // "BAT/USD",
  // "BCH/USD",
  // "CRV/USD",
  // "DOGE/USD",
  // "DOT/USD",
  // "GRT/USD",
  // "LINK/USD",
  // "MKR/USD",
  // "SOL/USD",
  // "SUSHI/USD",
  // "UNI/USD",
  // "USDC/USD",
  // "USDT/USD",
  // "XTZ/USD",
  // "YFI/USD",
}
