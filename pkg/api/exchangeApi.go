package api

import (
	"cryptoBot/pkg/constants"
	"cryptoBot/pkg/data/domains"
	"time"
)

type ExchangeApi interface {
	GetCurrentCoinPrice(coin *domains.Coin) (int64, error)
	GetKlines(coin *domains.Coin, interval string, limit int, fromTime time.Time) (KlinesDto, error)

	BuyCoinByMarket(coin *domains.Coin, amount float64, price int64) (OrderResponseDto, error)
	SellCoinByMarket(coin *domains.Coin, amount float64, price int64) (OrderResponseDto, error)

	OpenFuturesOrder(coin *domains.Coin, amount float64, price int64, futuresType constants.FuturesType) (OrderResponseDto, error)
	CloseFuturesOrder(coin *domains.Coin, openedTransaction *domains.Transaction, price int64) (OrderResponseDto, error)
}

type OrderResponseDto interface {
	CalculateAvgPrice() int64
	CalculateTotalCost() int64
	CalculateCommissionInUsd() int64
	GetAmount() float64
}

type KlinesDto interface {
	GetKlines() []KlineDto
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
