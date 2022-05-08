package api

import (
	"cryptoBot/pkg/constants"
	"cryptoBot/pkg/data/domains"
)

type ExchangeApi interface {
	GetCurrentCoinPrice(coin *domains.Coin) (int64, error)

	BuyCoinByMarket(coin *domains.Coin, amount float64, price int64) (OrderResponseDto, error)
	SellCoinByMarket(coin *domains.Coin, amount float64, price int64) (OrderResponseDto, error)

	OpenFuturesOrder(coin *domains.Coin, amount float64, futuresType constants.FuturesType, leverage int) (OrderResponseDto, error)
	CloseFuturesOrder(openedTransaction *domains.Transaction) (OrderResponseDto, error)
}

type OrderResponseDto interface {
	CalculateAvgPrice() int64
	CalculateTotalCost() int64
	CalculateCommissionInUsd() int64
	GetAmount() float64
}
