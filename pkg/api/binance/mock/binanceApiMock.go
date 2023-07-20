package mock

import (
	"cryptoBot/pkg/api"
	"cryptoBot/pkg/api/mock"
	"cryptoBot/pkg/constants/futureType"
	"cryptoBot/pkg/data/domains"
	"errors"
	"fmt"
	"time"
)

func NewBinanceApiMock() api.ExchangeApi {
	return &BinanceApiMock{}
}

type BinanceApiMock struct {
}

func (api *BinanceApiMock) GetKlines(coin *domains.Coin, interval string, limit int, fromTime time.Time) (api.KlinesDto, error) {
	return nil, errors.New("Not implemented for Binance API")
}

func (api *BinanceApiMock) GetKlinesFutures(coin *domains.Coin, interval string, limit int, fromTime time.Time) (api.KlinesDto, error) {
	return nil, errors.New("Not implemented for Binance API")
}

func (api *BinanceApiMock) OpenFuturesOrder(coin *domains.Coin, amount float64, price float64, futuresType futureType.FuturesType, stopLossPriceInCents float64) (api.OrderResponseDto, error) {
	return nil, errors.New("Futures api is not implemented")
}
func (api *BinanceApiMock) CloseFuturesOrder(coin *domains.Coin, openedTransaction *domains.Transaction, price float64) (api.OrderResponseDto, error) {
	return nil, errors.New("Futures api is not implemented")
}

func (api *BinanceApiMock) GetCurrentCoinPriceForFutures(coin *domains.Coin) (float64, error) {
	return 0, errors.New("Shouldn't be called.")
}

func (api *BinanceApiMock) GetCurrentCoinPrice(coin *domains.Coin) (float64, error) {
	return 0, errors.New("Shouldn't be called.")
}

func (api *BinanceApiMock) GetWalletBalance() (api.WalletBalanceDto, error) {
	return &mock.BalanceDtoMock{}, nil
}

func (api *BinanceApiMock) SetFuturesLeverage(coin *domains.Coin, leverage int) error {
	return nil
}

func (api *BinanceApiMock) SetIsolatedMargin(coin *domains.Coin, leverage int) error {
	return nil
}

func (api *BinanceApiMock) IsFuturesPositionOpened(coin *domains.Coin, openedOrder *domains.Transaction) bool {
	return true
}
func (api *BinanceApiMock) GetCloseTradeRecord(coin *domains.Coin, openTransaction *domains.Transaction) (api.OrderResponseDto, error) {
	return nil, nil
}

func (api *BinanceApiMock) GetLastFuturesOrder(coin *domains.Coin, clientOrderId string) (api.OrderResponseDto, error) {
	return nil, nil
}

func (api *BinanceApiMock) GetActiveFuturesConditionalOrder(coin *domains.Coin, conditionalOrder *domains.ConditionalOrder) (api.OrderResponseDto, error) {
	return nil, nil
}

var countOfNotSoldTransactions = 0
var maxCountOfNotSoldTransactions = 0

func (api *BinanceApiMock) BuyCoinByMarket(coin *domains.Coin, amount float64, price float64) (api.OrderResponseDto, error) {
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

func (api *BinanceApiMock) SellCoinByMarket(coin *domains.Coin, amount float64, price float64) (api.OrderResponseDto, error) {
	countOfNotSoldTransactions = countOfNotSoldTransactions - 1

	return &orderResponseMockDto{
		price:  price,
		amount: amount,
	}, nil
}

type orderResponseMockDto struct {
	price  float64
	amount float64
}

func (d *orderResponseMockDto) CalculateAvgPrice() float64 {
	return float64(d.price) * 0.01
}

func (d *orderResponseMockDto) CalculateTotalCost() float64 {
	return d.price * d.amount
}

func (d *orderResponseMockDto) CalculateCommissionInUsd() float64 {
	return float64(d.CalculateTotalCost()) * 0.001 // 0.1% commission
}

func (d *orderResponseMockDto) GetAmount() float64 {
	return d.amount
}

func (d *orderResponseMockDto) GetCreatedAt() *time.Time {
	return nil
}
