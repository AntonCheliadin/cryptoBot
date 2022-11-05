package constants

type TransactionType int8

const (
	BUY TransactionType = iota
	SELL
)

type TradingType int8

const (
	SPOT TradingType = iota
	FUTURES
)
