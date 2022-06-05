package analyser

import (
	"cryptoBot/pkg/api"
	"cryptoBot/pkg/constants"
	"cryptoBot/pkg/constants/bybit"
	"cryptoBot/pkg/data/domains"
	"cryptoBot/pkg/repository"
	"fmt"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"time"
)

var klinesFetcherServiceImpl *KlinesFetcherService

func NewKlinesFetcherService(exchangeApi api.ExchangeApi, klineRepo repository.Kline) *KlinesFetcherService {
	if movingAverageStrategyAnalyserServiceImpl != nil {
		panic("Unexpected try to create second service instance")
	}
	klinesFetcherServiceImpl = &KlinesFetcherService{
		klineRepo:   klineRepo,
		exchangeApi: exchangeApi,
	}
	return klinesFetcherServiceImpl
}

type KlinesFetcherService struct {
	klineRepo   repository.Kline
	exchangeApi api.ExchangeApi
}

func (s *KlinesFetcherService) FetchKlines(coin *domains.Coin, from string, to string) bool {
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

func (s *KlinesFetcherService) fetchKlinesForPeriod(coin *domains.Coin, from string, to string, interval string) error {
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

func (s *KlinesFetcherService) saveKlines(coin *domains.Coin, klinesDto api.KlinesDto) {
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
