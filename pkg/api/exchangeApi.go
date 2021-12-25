package api

import (
	"tradingBot/pkg/data/domains"
	"tradingBot/pkg/data/dto/binance"
)

type ExchangeApi interface {
	GetCurrentCoinPrice(coin *domains.Coin) (int64, error)

	BuyCoinByMarket(coin *domains.Coin, amount float64) (*binance.OrderResponseDto, error)
	SellCoinByMarket(coin *domains.Coin, amount float64) (*binance.OrderResponseDto, error)
}
