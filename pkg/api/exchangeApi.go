package api

import (
	"cryptoBot/pkg/constants/futureType"
	"cryptoBot/pkg/data/domains"
	"time"
)

type ExchangeApi interface {
	GetCurrentCoinPriceForFutures(coin *domains.Coin) (float64, error)
	GetCurrentCoinPrice(coin *domains.Coin) (float64, error)
	GetKlines(coin *domains.Coin, interval string, limit int, fromTime time.Time) (KlinesDto, error)
	GetKlinesFutures(coin *domains.Coin, interval string, limit int, fromTime time.Time) (KlinesDto, error)

	BuyCoinByMarket(coin *domains.Coin, amount float64, price float64) (OrderResponseDto, error)
	SellCoinByMarket(coin *domains.Coin, amount float64, price float64) (OrderResponseDto, error)

	OpenFuturesOrder(coin *domains.Coin, amount float64, price float64, futuresType futureType.FuturesType, stopLossPriceInCents float64) (OrderResponseDto, error)
	CloseFuturesOrder(coin *domains.Coin, openedTransaction *domains.Transaction, price float64) (OrderResponseDto, error)
	IsFuturesPositionOpened(coin *domains.Coin, openedOrder *domains.Transaction) bool
	GetCloseTradeRecord(coin *domains.Coin, openTransaction *domains.Transaction) (OrderResponseDto, error)
	GetLastFuturesOrder(coin *domains.Coin, clientOrderId string) (OrderResponseDto, error)

	//OpenFuturesConditionalOrder(coin *domains.Coin, amount float64, price int64, futuresType futureType.FuturesType, stopLoss bool, takeProfit bool)
	GetActiveFuturesConditionalOrder(coin *domains.Coin, conditionalOrder *domains.ConditionalOrder) (OrderResponseDto, error)

	GetWalletBalance() (WalletBalanceDto, error)
	SetFuturesLeverage(coin *domains.Coin, leverage int) error
	SetIsolatedMargin(coin *domains.Coin, leverage int) error
}

type OrderResponseDto interface {
	CalculateAvgPrice() float64
	CalculateTotalCost() float64
	CalculateCommissionInUsd() float64
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
	GetOpen() float64
	GetHigh() float64
	GetLow() float64
	GetClose() float64
}

type WalletBalanceDto interface {
	GetAvailableBalanceInCents() float64
}
