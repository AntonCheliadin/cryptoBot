package bybit

import (
	"cryptoBot/pkg/api"
	"cryptoBot/pkg/constants"
	"cryptoBot/pkg/data/domains"
	"errors"
)

func NewBybitApi() api.ExchangeApi {
	return &BybitApi{}
}

//todo BybitApi
type BybitApi struct {
}

func (api *BybitApi) GetBars(coin *domains.Coin) (int64, error) {
	return 0, errors.New("Not implemented")
}

func (api *BybitApi) GetCurrentCoinPrice(coin *domains.Coin) (int64, error) {
	return 0, errors.New("Not implemented")
}

func (api *BybitApi) BuyCoinByMarket(coin *domains.Coin, amount float64, price int64) (api.OrderResponseDto, error) {
	return nil, errors.New("Not implemented")
}

func (api *BybitApi) SellCoinByMarket(coin *domains.Coin, amount float64, price int64) (api.OrderResponseDto, error) {
	return nil, errors.New("Not implemented")
}

func (api *BybitApi) OpenFuturesOrder(coin *domains.Coin, amount float64, futuresType constants.FuturesType, leverage int) (api.OrderResponseDto, error) {
	return nil, errors.New("Futures api is not implemented")
}
func (api *BybitApi) CloseFuturesOrder(openedTransaction *domains.Transaction) (api.OrderResponseDto, error) {
	return nil, errors.New("Futures api is not implemented")
}
