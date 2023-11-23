package mock

import (
	"cryptoBot/pkg/api"
	"cryptoBot/pkg/api/mock"
	"cryptoBot/pkg/constants/futureType"
	"cryptoBot/pkg/data/domains"
	"cryptoBot/pkg/data/dto/bybit"
	"cryptoBot/pkg/util"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

func NewBybitApiMock() api.ExchangeApi {
	return &BybitApiMock{}
}

type BybitApiMock struct {
}

func (api *BybitApiMock) GetKlines(coin *domains.Coin, interval string, limit int, fromTime time.Time) (api.KlinesDto, error) {
	resp, err := http.Get("https://api.bytick.com/public/linear/kline?" +
		"symbol=" + coin.Symbol +
		"&interval=" + interval +
		"&limit=" + strconv.Itoa(limit) +
		"&from=" + strconv.Itoa(util.GetSecondsByTime(fromTime)))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var dto bybit.KlinesDto
	if err := json.NewDecoder(resp.Body).Decode(&dto); err != nil {
		return nil, err
	}

	return &dto, nil
}

func (api *BybitApiMock) GetKlinesFutures(coin *domains.Coin, interval string, limit int, fromTime time.Time) (api.KlinesDto, error) {
	return nil, errors.New("Not implemented for Bybit API mock")
}

func (api *BybitApiMock) OpenFuturesOrder(coin *domains.Coin, amount float64, price float64, futuresType futureType.FuturesType, stopLossPriceInCents float64) (api.OrderResponseDto, error) {
	return &orderResponseMockDto{
		price:  price,
		amount: amount,
	}, nil
}
func (api *BybitApiMock) CloseFuturesOrder(coin *domains.Coin, openedTransaction *domains.Transaction, price float64) (api.OrderResponseDto, error) {
	return &orderResponseMockDto{
		price:  price,
		amount: openedTransaction.Amount,
	}, nil
}

func (api *BybitApiMock) GetCurrentCoinPriceForFutures(coin *domains.Coin) (float64, error) {
	return 0, errors.New("Shouldn't be called.")
}

func (api *BybitApiMock) GetCurrentCoinPrice(coin *domains.Coin) (float64, error) {
	return 0, errors.New("Shouldn't be called.")
}

var countOfNotSoldTransactions = 0
var maxCountOfNotSoldTransactions = 0

func (api *BybitApiMock) BuyCoinByMarket(coin *domains.Coin, amount float64, price float64) (api.OrderResponseDto, error) {
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

func (api *BybitApiMock) SellCoinByMarket(coin *domains.Coin, amount float64, price float64) (api.OrderResponseDto, error) {
	countOfNotSoldTransactions = countOfNotSoldTransactions - 1

	return &orderResponseMockDto{
		price:  price,
		amount: amount,
	}, nil
}

func (api *BybitApiMock) GetWalletBalance() (api.WalletBalanceDto, error) {
	return &mock.BalanceDtoMock{}, nil
}

func (api *BybitApiMock) SetFuturesLeverage(coin *domains.Coin, leverage int) error {
	return nil
}

func (api *BybitApiMock) SetIsolatedMargin(coin *domains.Coin, leverage int) error {
	return nil
}

func (api *BybitApiMock) IsFuturesPositionOpened(coin *domains.Coin, openedOrder *domains.Transaction) bool {
	return true
}
func (api *BybitApiMock) GetCloseTradeRecord(coin *domains.Coin, openTransaction *domains.Transaction) (api.OrderResponseDto, error) {
	return nil, nil
}

func (api *BybitApiMock) GetLastFuturesOrder(coin *domains.Coin, clientOrderId string) (api.OrderResponseDto, error) {
	return nil, nil
}
func (api *BybitApiMock) GetActiveFuturesConditionalOrder(coin *domains.Coin, conditionalOrder *domains.ConditionalOrder) (api.OrderResponseDto, error) {
	return nil, nil
}

type orderResponseMockDto struct {
	price  float64
	amount float64
}

func (d *orderResponseMockDto) CalculateAvgPrice() float64 {
	return float64(d.price)
}

func (d *orderResponseMockDto) CalculateTotalCost() float64 {
	return d.price * d.amount
}

func (d *orderResponseMockDto) CalculateCommissionInUsd() float64 {
	return float64(d.CalculateTotalCost()) * 0.00055 // 0.055% for maker
}

func (d *orderResponseMockDto) GetAmount() float64 {
	return d.amount
}
func (d *orderResponseMockDto) GetCreatedAt() *time.Time {
	return nil
}
