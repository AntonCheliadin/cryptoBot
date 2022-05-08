package analyser

import (
	"cryptoBot/pkg/api"
	"cryptoBot/pkg/constants"
	"cryptoBot/pkg/data/domains"
	"cryptoBot/pkg/repository"
	"cryptoBot/pkg/service/date"
	"cryptoBot/pkg/service/trading"
	"github.com/spf13/viper"
	"time"
)

var movingAverageStrategyAnalyserServiceImpl *MovingAverageStrategyAnalyserService

func NewMovingAverageStrategyAnalyserService(transactionRepo repository.Transaction, priceChangeRepo repository.PriceChange,
	exchangeApi api.ExchangeApi, tradingService *trading.MovingAverageStrategyTradingService) *MovingAverageStrategyAnalyserService {
	if movingAverageStrategyAnalyserServiceImpl != nil {
		panic("Unexpected try to create second service instance")
	}
	movingAverageStrategyAnalyserServiceImpl = &MovingAverageStrategyAnalyserService{
		transactionRepo: transactionRepo,
		priceChangeRepo: priceChangeRepo,
		exchangeApi:     exchangeApi,
		tradingService:  tradingService,
	}
	return movingAverageStrategyAnalyserServiceImpl
}

type MovingAverageStrategyAnalyserService struct {
	transactionRepo repository.Transaction
	priceChangeRepo repository.PriceChange
	exchangeApi     api.ExchangeApi
	tradingService  *trading.MovingAverageStrategyTradingService
}

func (s *MovingAverageStrategyAnalyserService) AnalyseCoin(coin *domains.Coin, from string, to string) {
	//todo fetch all candles for a long period

	candleDuration := time.Duration(viper.GetInt64("strategy.ma.interval"))
	timeMax, _ := time.Parse(constants.DATE_FORMAT, to)
	timeIterator, _ := time.Parse(constants.DATE_FORMAT, from)

	for ; timeIterator.Before(timeMax); timeIterator = timeIterator.Add(time.Minute * candleDuration) {
		clockMock := date.GetClockMock(timeIterator)
		s.tradingService.Clock = clockMock
		s.tradingService.ExchangeDataService.Clock = clockMock

		s.tradingService.BotSingleAction(coin)
	}
}
