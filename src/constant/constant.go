
package constant

import (
  "net/http"
)


const CHANNEL_BUFFER_SIZE int = 10
const ORDER_SIZE_USD float64 = 500
const ENDPOINT = "https://paper-api.alpaca.markets/v2"
var AUTH_HEADERS http.Header
var KEY string
var SECRET string


var STOCK_LIST = []string{
  "AAPL",
  "MSFT",
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
  "SHIB/USD",
  // "AAVE/USD",
  // "AVAX/USD",
  // "BAT/USD",
  // "BCH/USD",
  // "CRV/USD",
  // "DOGE/USD",
  // "DOT/USD",
  // "ETH/USD",
  // "GRT/USD",
  // "LINK/USD",
  // "LTC/USD",
  // "MKR/USD",
  // "SOL/USD",
  // "SUSHI/USD",
  // "UNI/USD",
  // "USDC/USD",
  // "USDT/USD",
  // "XTZ/USD",
  // "YFI/USD",
}
