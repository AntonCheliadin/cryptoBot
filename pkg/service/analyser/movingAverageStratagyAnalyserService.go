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

var movingAverageStrategyAnalyserServiceImpl *MovingAverageStrategyAnalyserService

func NewMovingAverageStrategyAnalyserService(transactionRepo repository.Transaction, priceChangeRepo repository.PriceChange,
	exchangeApi api.ExchangeApi, tradingService *trading.MovingAverageStrategyTradingService,
	klineRepo repository.Kline) *MovingAverageStrategyAnalyserService {
	if movingAverageStrategyAnalyserServiceImpl != nil {
		panic("Unexpected try to create second service instance")
	}
	movingAverageStrategyAnalyserServiceImpl = &MovingAverageStrategyAnalyserService{
		klineRepo:       klineRepo,
		transactionRepo: transactionRepo,
		priceChangeRepo: priceChangeRepo,
		exchangeApi:     exchangeApi,
		tradingService:  tradingService,
	}
	return movingAverageStrategyAnalyserServiceImpl
}

type MovingAverageStrategyAnalyserService struct {
	klineRepo       repository.Kline
	transactionRepo repository.Transaction
	priceChangeRepo repository.PriceChange
	exchangeApi     api.ExchangeApi
	tradingService  *trading.MovingAverageStrategyTradingService
}

func (s *MovingAverageStrategyAnalyserService) AnalyseCoin(coin *domains.Coin, from string, to string) {
	timeMax, _ := time.Parse(constants.DATE_FORMAT, to)
	timeIterator, _ := time.Parse(constants.DATE_FORMAT, from)
	timeIterator = timeIterator.Add(time.Second * 2).Add(time.Hour)

	for ; timeIterator.Before(timeMax); timeIterator = timeIterator.Add(time.Minute * 15) {
		clockMock := date.NewClockMock(timeIterator)
		s.tradingService.Clock = clockMock
		s.tradingService.ExchangeDataService.Clock = clockMock
		s.tradingService.MovingAverageService.Clock = clockMock
		s.tradingService.StandardDeviationService.Clock = clockMock

		s.tradingService.BotSingleAction(coin)
	}
}

func (s *MovingAverageStrategyAnalyserService) saveKlines(coin *domains.Coin, klinesDto api.KlinesDto) {
	for _, dto := range klinesDto.GetKlines() {
		if existedKline, _ := s.klineRepo.FindOpenedAtMoment(coin.Id, dto.GetStartAt(), dto.GetInterval()); existedKline == nil {
			kline := domains.Kline{
				CoinId:    coin.Id,
				OpenTime:  dto.GetStartAt(),
				CloseTime: dto.GetCloseAt(),
				Interval:  dto.GetInterval(),
				Open:      dto.GetOpen(),
				High:      dto.GetHigh(),
				Low:       dto.GetLow(),
				Close:     dto.GetClose(),
			}

			_ = s.klineRepo.SaveKline(&kline)
		}
	}
}
