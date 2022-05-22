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
	//if err := s.fetchKlinesForPeriod(coin, from, to); err != nil {
	//	zap.S().Errorf("Error during fetchKlinesForPeriod %s", err.Error())
	//	return
	//}

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

func (s *MovingAverageStrategyAnalyserService) fetchKlinesForPeriod(coin *domains.Coin, from string, to string) error {
	timeFrom, _ := time.Parse(constants.DATE_FORMAT, from)
	timeTo, _ := time.Parse(constants.DATE_FORMAT, to)

	timeIter := timeFrom
	for timeIter.Before(timeTo) {
		klinesDto, err := s.exchangeApi.GetKlines(coin, viper.GetString("strategy.ma.interval"), bybit.BYBIT_MAX_LIMIT, timeFrom)
		if err != nil {
			zap.S().Errorf("Error on GetCurrentCoinPrice: %s", err)
			return err
		}
		fmt.Printf("\nklinesDto=%s\n", klinesDto)

		s.saveKlines(coin, klinesDto)

		klineLength := len(klinesDto.GetKlines())
		lastKline := klinesDto.GetKlines()[klineLength-1]
		timeIter = lastKline.GetCloseAt()
	}

	return nil
}

func (s *MovingAverageStrategyAnalyserService) saveKlines(coin *domains.Coin, klinesDto api.KlinesDto) {
	for _, dto := range klinesDto.GetKlines() {
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
