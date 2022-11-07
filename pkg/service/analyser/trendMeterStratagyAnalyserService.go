package analyser

import (
	"cryptoBot/pkg/constants"
	"cryptoBot/pkg/data/domains"
	"cryptoBot/pkg/service/date"
	"cryptoBot/pkg/service/trading"
	"time"
)

var trendMeterStratagyAnalyserServiceImpl *TrendMeterStratagyAnalyserService

func NewTrendMeterStratagyAnalyserService(tradingService *trading.TrendMeterStrategyTradingService) *TrendMeterStratagyAnalyserService {
	if trendMeterStratagyAnalyserServiceImpl != nil {
		panic("Unexpected try to create second service instance")
	}
	trendMeterStratagyAnalyserServiceImpl = &TrendMeterStratagyAnalyserService{
		tradingService: tradingService,
	}
	return trendMeterStratagyAnalyserServiceImpl
}

type TrendMeterStratagyAnalyserService struct {
	tradingService *trading.TrendMeterStrategyTradingService
}

func (s *TrendMeterStratagyAnalyserService) AnalyseCoin(coin *domains.Coin, from string, to string) {
	timeMax, _ := time.Parse(constants.DATE_FORMAT, to)
	timeIterator, _ := time.Parse(constants.DATE_FORMAT, from)
	timeIterator = timeIterator.Add(time.Second * 2).Add(time.Hour)

	for ; timeIterator.Before(timeMax); timeIterator = timeIterator.Add(time.Minute * 15) {
		clockMock := date.GetClockMock(timeIterator)

		s.tradingService.Clock = clockMock
		s.tradingService.StandardDeviationService.TechanConvertorService.Clock = clockMock
		s.tradingService.ExchangeDataService.Clock = clockMock
		s.tradingService.StandardDeviationService.Clock = clockMock
		s.tradingService.OrderManagerService.ProfitLossFinderService.Clock = clockMock
		s.tradingService.OrderManagerService.Clock = clockMock

		s.tradingService.BotActionBuyMoreIfNeeded(coin)
		s.tradingService.BotActionCloseOrderIfNeeded(coin)
		s.tradingService.BotActionOpenOrderIfNeeded(coin)
	}
}
