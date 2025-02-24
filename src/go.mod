module github.com/Kjellemann1/AlgoTrader-Go

go 1.23.6

replace github.com/Kjellemann1/AlgoTrader-Go/src => ../src

require (
	github.com/go-sql-driver/mysql v1.9.0
	github.com/gorilla/websocket v1.5.3
	github.com/joho/godotenv v1.5.1
	github.com/qdm12/reprint v0.0.0-20200326205758-722754a53494
	github.com/shopspring/decimal v1.4.0
	github.com/stretchr/testify v1.10.0
	github.com/valyala/fastjson v1.6.4
)

require (
	filippo.io/edwards25519 v1.1.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
