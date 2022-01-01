package api

import (
	"tradingBot/pkg/data/domains"
)

type OrderDto interface {
	CalculateAvgPrice() int64
	CalculateTotalCost() int64
	CalculateCommissionInUsd() int64
	GetAmount() float64
}

type ExchangeApi interface {
	GetCurrentCoinPrice(coin *domains.Coin) (int64, error)

	BuyCoinByMarket(coin *domains.Coin, amount float64) (OrderDto, error)
	SellCoinByMarket(coin *domains.Coin, amount float64) (OrderDto, error)
}
