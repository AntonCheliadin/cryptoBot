package analyser

import (
	"cryptoBot/pkg/api"
	"cryptoBot/pkg/constants"
	"cryptoBot/pkg/data/domains"
	"cryptoBot/pkg/repository"
	"cryptoBot/pkg/service/date"
	"cryptoBot/pkg/service/trading"
	"time"
)

var movingAverageResistanceStratagyAnalyserServiceImpl *MovingAverageResistanceStratagyAnalyserService

func NewMovingAverageResistanceStratagyAnalyserService(transactionRepo repository.Transaction, priceChangeRepo repository.PriceChange,
	exchangeApi api.ExchangeApi, tradingService *trading.MovingAverageResistanceStrategyTradingService,
	klineRepo repository.Kline) *MovingAverageResistanceStratagyAnalyserService {
	if movingAverageResistanceStratagyAnalyserServiceImpl != nil {
		panic("Unexpected try to create second service instance")
	}
	movingAverageResistanceStratagyAnalyserServiceImpl = &MovingAverageResistanceStratagyAnalyserService{
		klineRepo:       klineRepo,
		transactionRepo: transactionRepo,
		priceChangeRepo: priceChangeRepo,
		exchangeApi:     exchangeApi,
		tradingService:  tradingService,
	}
	return movingAverageResistanceStratagyAnalyserServiceImpl
}

type MovingAverageResistanceStratagyAnalyserService struct {
	klineRepo       repository.Kline
	transactionRepo repository.Transaction
	priceChangeRepo repository.PriceChange
	exchangeApi     api.ExchangeApi
	tradingService  *trading.MovingAverageResistanceStrategyTradingService
}

func (s *MovingAverageResistanceStratagyAnalyserService) AnalyseCoin(coin *domains.Coin, from string, to string) {
	timeMax, _ := time.Parse(constants.DATE_FORMAT, to)
	timeIterator, _ := time.Parse(constants.DATE_FORMAT, from)
	timeIterator = timeIterator.Add(time.Second * 2).Add(time.Hour)

	for ; timeIterator.Before(timeMax); timeIterator = timeIterator.Add(time.Minute * 15) {
		clockMock := date.GetClockMock(timeIterator)
		s.tradingService.Clock = clockMock
		s.tradingService.ExchangeDataService.Clock = clockMock
		s.tradingService.MovingAverageService.Clock = clockMock

		s.tradingService.BotSingleAction(coin)
	}
}
