package exchange

import (
	"cryptoBot/pkg/api"
	"cryptoBot/pkg/data/domains"
	"cryptoBot/pkg/repository"
	"cryptoBot/pkg/service/date"
	"cryptoBot/pkg/util"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"strconv"
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

//Deprecated: use GetCurrentPriceWithInterval instead
func (s *DataService) GetCurrentPrice(coin *domains.Coin) (float64, error) {
	interval := viper.GetInt("strategy.trendMeter.interval")

	return s.GetCurrentPriceWithInterval(coin, interval)
}

func (s *DataService) GetCurrentPriceWithInterval(coin *domains.Coin, interval int) (float64, error) {
	if s.Clock.NowTime().Minute()%interval == 0 {
		strategyIntervalString := strconv.Itoa(interval)
		if kline, _ := s.klineRepo.FindOpenedAtMoment(coin.Id, util.RoundToMinutes(s.Clock.NowTime()), strategyIntervalString); kline != nil {
			return kline.Open, nil
		}
	}

	currentCoinPrice, err := s.exchangeApi.GetCurrentCoinPrice(coin)
	if err != nil {
		zap.S().Errorf("Error during GetCurrentCoinPrice at %s (rounded to %s) - %s", s.Clock.NowTime(), util.RoundToMinutes(s.Clock.NowTime()), err.Error())
	}
	return currentCoinPrice, err
}

func (s *DataService) GetCurrentPriceForFutures(coin *domains.Coin, interval int) (float64, error) {
	if s.Clock.NowTime().Minute()%interval == 0 {
		strategyIntervalString := strconv.Itoa(interval)
		if kline, _ := s.klineRepo.FindOpenedAtMoment(coin.Id, util.RoundToMinutes(s.Clock.NowTime()), strategyIntervalString); kline != nil {
			return kline.Open, nil
		}
	}

	currentCoinPrice, err := s.exchangeApi.GetCurrentCoinPriceForFutures(coin)
	if err != nil {
		zap.S().Errorf("Error during GetCurrentPriceForFutures at %s (rounded to %s) - %s", s.Clock.NowTime(), util.RoundToMinutes(s.Clock.NowTime()), err.Error())
	}
	return currentCoinPrice, err
}

func (s *DataService) IsPositionOpened(coin *domains.Coin, openedOrder *domains.Transaction) bool {
	return s.exchangeApi.IsFuturesPositionOpened(coin, openedOrder)
}
