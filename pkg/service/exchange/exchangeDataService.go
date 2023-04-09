package exchange

import (
	"cryptoBot/pkg/api"
	"cryptoBot/pkg/data/domains"
	"cryptoBot/pkg/repository"
	"cryptoBot/pkg/service/date"
	"cryptoBot/pkg/util"
	"github.com/spf13/viper"
	"go.uber.org/zap"
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
	if s.Clock.NowTime().Minute()%viper.GetInt("strategy.trendMeter.interval") == 0 {
		strategyInterval := viper.GetString("strategy.trendMeter.interval")
		if kline, _ := s.klineRepo.FindOpenedAtMoment(coin.Id, util.RoundToMinutes(s.Clock.NowTime()), strategyInterval); kline != nil {
			return kline.Open, nil
		}
	}

	if kline, _ := s.klineRepo.FindOpenedAtMoment(coin.Id, util.RoundToMinutes(s.Clock.NowTime()), "60"); kline != nil {
		return kline.Open, nil
	}

	currentCoinPrice, err := s.exchangeApi.GetCurrentCoinPrice(coin)
	if err != nil {
		zap.S().Errorf("Error during GetCurrentCoinPrice at %s (rounded to %s) - %s", s.Clock.NowTime(), util.RoundToMinutes(s.Clock.NowTime()), err.Error())
	}
	return currentCoinPrice, err
}

func (s *DataService) IsPositionOpened(coin *domains.Coin, openedOrder *domains.Transaction) bool {
	return s.exchangeApi.IsFuturesPositionOpened(coin, openedOrder)
}
