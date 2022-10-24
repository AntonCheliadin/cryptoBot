package api

import (
	"cryptoBot/pkg/constants/futureType"
	"cryptoBot/pkg/data/domains"
	"time"
)

type ExchangeApi interface {
	GetCurrentCoinPrice(coin *domains.Coin) (int64, error)
	GetKlines(coin *domains.Coin, interval string, limit int, fromTime time.Time) (KlinesDto, error)

	BuyCoinByMarket(coin *domains.Coin, amount float64, price int64) (OrderResponseDto, error)
	SellCoinByMarket(coin *domains.Coin, amount float64, price int64) (OrderResponseDto, error)

	OpenFuturesOrder(coin *domains.Coin, amount float64, price int64, futuresType futureType.FuturesType, stopLossPriceInCents int64) (OrderResponseDto, error)
	CloseFuturesOrder(coin *domains.Coin, openedTransaction *domains.Transaction, price int64) (OrderResponseDto, error)
	IsFuturesPositionOpened(coin *domains.Coin, openedOrder *domains.Transaction) bool
	GetCloseTradeRecord(coin *domains.Coin, openTransaction *domains.Transaction) (OrderResponseDto, error)
	GetLastFuturesOrder(coin *domains.Coin, clientOrderId string) (OrderResponseDto, error)

	//OpenFuturesConditionalOrder(coin *domains.Coin, amount float64, price int64, futuresType futureType.FuturesType, stopLoss bool, takeProfit bool)
	GetActiveFuturesConditionalOrder(coin *domains.Coin, conditionalOrder *domains.ConditionalOrder) (OrderResponseDto, error)

	GetWalletBalance() (WalletBalanceDto, error)
	SetFuturesLeverage(coin *domains.Coin, leverage int) error
}

type OrderResponseDto interface {
	CalculateAvgPrice() int64
	CalculateTotalCost() int64
	CalculateCommissionInUsd() int64
	GetAmount() float64
	GetCreatedAt() *time.Time
}

type KlinesDto interface {
	GetKlines() []KlineDto
	String() string
}

type KlineDto interface {
	GetSymbol() string
	GetInterval() string
	GetStartAt() time.Time
	GetCloseAt() time.Time
	GetOpen() int64
	GetHigh() int64
	GetLow() int64
	GetClose() int64
}

type WalletBalanceDto interface {
	GetAvailableBalanceInCents() int64
}
