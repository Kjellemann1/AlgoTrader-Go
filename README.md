# AlgoTrader-Go

An algorithmic trading system written in Go, built to run automated strategies against the Alpaca paper trading API. It handles real-time market data ingestion, order execution, position management, and persistent state tracking across both stocks and cryptocurrencies.

## Overview

The system connects to Alpaca's WebSocket streams for live market data (1-minute OHLC bars and individual trades) and account updates (order fills, cancellations). It maintains rolling price windows per asset and runs configurable trading strategies that generate buy/sell signals based on incoming data. Orders are sent via Alpaca's REST API.

The core loop looks roughly like this: market data comes in through WebSocket, gets routed to the relevant asset, updates its price history, triggers strategy evaluation, and if a signal fires, an order is placed. A separate WebSocket connection listens for order fills and updates position state accordingly. All trades and positions are persisted to a MySQL database.

## Architecture

The system runs several concurrent goroutines coordinated through Go's context and sync primitives:

- **Market listeners** maintain WebSocket connections to Alpaca's data streams (separate feeds for stocks and crypto), parse incoming messages, and distribute them to a worker pool for processing.
- **Account listener** tracks order fills and trade updates in real time, reconciles local state with the broker on reconnection, and handles edge cases like orders that filled while the connection was down.
- **Asset management** keeps per-asset state including rolling OHLC windows (500 bars), active positions, and strategy channels. Each asset runs its strategy evaluation in its own goroutine.
- **Database worker** receives trade and position updates through a buffered channel and writes them to MySQL asynchronously, decoupling persistence from the trading loop.
- **Shutdown handler** supports graceful termination with options to save state, liquidate positions, or abort.

## Key features

- WebSocket reconnection with exponential backoff
- Order retry logic with rate limit handling (HTTP 429)
- Position reconciliation on reconnect (queries broker for missed fills)
- Stop loss, take profit, and trailing stop risk controls
- Push notifications via Pushover for errors and trade alerts
- Precise decimal arithmetic using shopspring/decimal to avoid floating-point issues in financial calculations

## Dependencies

- [gorilla/websocket](https://github.com/gorilla/websocket) for real-time data streams
- [fastjson](https://github.com/valyala/fastjson) for high-throughput JSON parsing
- [shopspring/decimal](https://github.com/shopspring/decimal) for precise financial arithmetic
- [go-sql-driver/mysql](https://github.com/go-sql-driver/mysql) for persistence
- [godotenv](https://github.com/joho/godotenv) for configuration
- [testify](https://github.com/stretchr/testify) for testing

## Configuration

The system reads its configuration from environment variables (loaded from a `.env` file): Alpaca API credentials, MySQL connection details, Pushover notification keys, and log output paths. Trading parameters like position sizing, retry limits, and window sizes are defined as constants.

## Note

This runs against Alpaca's paper trading environment only.
