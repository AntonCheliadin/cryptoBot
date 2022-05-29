package analyser

import (
	"cryptoBot/pkg/api"
	"cryptoBot/pkg/constants"
	"cryptoBot/pkg/constants/bybit"
	"cryptoBot/pkg/data/domains"
	"cryptoBot/pkg/repository"
	"cryptoBot/pkg/service/date"
	"cryptoBot/pkg/service/trading"
	"fmt"
	"github.com/spf13/viper"
	"go.uber.org/zap"
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

	for ; timeIterator.Before(timeMax); timeIterator = timeIterator.Add(time.Minute) {
		clockMock := date.GetClockMock(timeIterator)
		s.tradingService.Clock = clockMock
		s.tradingService.ExchangeDataService.Clock = clockMock

		s.tradingService.BotSingleAction(coin)
	}
}

func (s *MovingAverageStrategyAnalyserService) FetchKlines(coin *domains.Coin, from string, to string) bool {
	if err := s.fetchKlinesForPeriod(coin, from, to, viper.GetString("strategy.ma.interval")); err != nil {
		zap.S().Errorf("Error during fetchKlinesForPeriod %s", err.Error())
		return true
	}

	if err := s.fetchKlinesForPeriod(coin, from, to, "1"); err != nil {
		zap.S().Errorf("Error during fetchKlinesForPeriod %s", err.Error())
		return true
	}
	return false
}

func (s *MovingAverageStrategyAnalyserService) fetchKlinesForPeriod(coin *domains.Coin, from string, to string, interval string) error {
	timeFrom, _ := time.Parse(constants.DATE_FORMAT, from)
	timeTo, _ := time.Parse(constants.DATE_FORMAT, to)

	timeIter := timeFrom
	for timeIter.Before(timeTo) {
		klinesDto, err := s.exchangeApi.GetKlines(coin, interval, bybit.BYBIT_MAX_LIMIT, timeIter)
		if err != nil {
			zap.S().Errorf("Error on fetch klines: %s", err)
			return err
		}
		fmt.Printf("Fetched %v klines from %v\n", len(klinesDto.GetKlines()), timeIter)

		s.saveKlines(coin, klinesDto)

		klineLength := len(klinesDto.GetKlines())
		lastKline := klinesDto.GetKlines()[klineLength-1]
		timeIter = lastKline.GetCloseAt()
	}

	return nil
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
