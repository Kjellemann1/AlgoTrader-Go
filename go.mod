module github.com/Kjellemann1/AlgoTrader-Go

go 1.23.5

toolchain go1.23.6

replace github.com/Kjellemann1/Gostuff => /home/kjellarne/Programming/GoStuff

require (
	github.com/Kjellemann1/Gostuff v0.0.0-00010101000000-000000000000
	github.com/go-sql-driver/mysql v1.8.1
	github.com/gorilla/websocket v1.5.3
	github.com/joho/godotenv v1.5.1
	github.com/markcheno/go-talib v0.0.0-20250114000313-ec55a20c902f
	github.com/shopspring/decimal v1.4.0
	github.com/stretchr/testify v1.9.0
	github.com/valyala/fastjson v1.6.4
)

require (
	filippo.io/edwards25519 v1.1.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
