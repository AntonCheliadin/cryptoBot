package mock

import (
	"cryptoBot/pkg/api"
	"cryptoBot/pkg/data/domains"
	"errors"
	"fmt"
)

func NewBinanceApiMock() api.ExchangeApi {
	return &BinanceApiMock{}
}

type BinanceApiMock struct {
}

func (api *BinanceApiMock) GetCurrentCoinPrice(coin *domains.Coin) (int64, error) {
	return 0, errors.New("Shouldn't be called.")
}

var countOfNotSoldTransactions = 0
var maxCountOfNotSoldTransactions = 0

func (api *BinanceApiMock) BuyCoinByMarket(coin *domains.Coin, amount float64, price int64) (api.OrderResponseDto, error) {
	countOfNotSoldTransactions = countOfNotSoldTransactions + 1

	if countOfNotSoldTransactions > maxCountOfNotSoldTransactions {
		maxCountOfNotSoldTransactions = countOfNotSoldTransactions
		fmt.Printf("------------maxCountOfNotSoldTransactions=%v \n", maxCountOfNotSoldTransactions)
	}

	return &orderResponseMockDto{
		price:  price,
		amount: amount,
	}, nil
}

func (api *BinanceApiMock) SellCoinByMarket(coin *domains.Coin, amount float64, price int64) (api.OrderResponseDto, error) {
	countOfNotSoldTransactions = countOfNotSoldTransactions - 1

	return &orderResponseMockDto{
		price:  price,
		amount: amount,
	}, nil
}

type orderResponseMockDto struct {
	price  int64
	amount float64
}

func (d *orderResponseMockDto) CalculateAvgPrice() int64 {
	return d.price
}

func (d *orderResponseMockDto) CalculateTotalCost() int64 {
	return int64(float64(d.price) * d.amount)
}

func (d *orderResponseMockDto) CalculateCommissionInUsd() int64 {
	return int64(float64(d.CalculateTotalCost()) * 0.001) // 0.1% commission
}

func (d *orderResponseMockDto) GetAmount() float64 {
	return d.amount
}
