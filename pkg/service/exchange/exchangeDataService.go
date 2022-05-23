package exchange

import (
	"cryptoBot/pkg/api"
	"cryptoBot/pkg/data/domains"
	"cryptoBot/pkg/repository"
	"cryptoBot/pkg/service/date"
	"github.com/spf13/viper"
)

var exchangeDataServiceImpl *DataService

func NewExchangeDataService(transactionRepo repository.Transaction, coinRepo repository.Coin, exchangeApi api.ExchangeApi,
	clock date.Clock, klineRepo repository.Kline) *DataService {
	if exchangeDataServiceImpl != nil {
		panic("Unexpected try to create second service instance")
	}
	exchangeDataServiceImpl = &DataService{
		transactionRepo: transactionRepo,
		coinRepo:        coinRepo,
		exchangeApi:     exchangeApi,
		Clock:           clock,
		klineRepo:       klineRepo,
	}
	return exchangeDataServiceImpl
}

type DataService struct {
	transactionRepo repository.Transaction
	coinRepo        repository.Coin
	exchangeApi     api.ExchangeApi
	Clock           date.Clock
	klineRepo       repository.Kline
}

func (s *DataService) GetCurrentPrice(coin *domains.Coin) (int64, error) {
	momentKline, _ := s.klineRepo.FindOpenedAtMoment(coin.Id, s.Clock.NowTime(), viper.GetString("strategy.ma.interval"))
	if momentKline != nil {
		return momentKline.Open, nil
	}

	return s.exchangeApi.GetCurrentCoinPrice(coin)
}
