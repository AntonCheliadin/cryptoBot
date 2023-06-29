package exchange

import (
	"cryptoBot/pkg/api"
	"cryptoBot/pkg/constants"
	"cryptoBot/pkg/constants/bybit"
	"cryptoBot/pkg/data/domains"
	"cryptoBot/pkg/repository"
	"cryptoBot/pkg/service/date"
	"errors"
	"fmt"
	"go.uber.org/zap"
	"time"
)

var klinesFetcherServiceImpl *KlinesFetcherService

func NewKlinesFetcherService(exchangeApi api.ExchangeApi, klineRepo repository.Kline, clock date.Clock) *KlinesFetcherService {
	klinesFetcherServiceImpl = &KlinesFetcherService{
		klineRepo:   klineRepo,
		exchangeApi: exchangeApi,
		Clock:       clock,
	}
	return klinesFetcherServiceImpl
}

type KlinesFetcherService struct {
	klineRepo   repository.Kline
	exchangeApi api.ExchangeApi
	Clock       date.Clock
}

func (s *KlinesFetcherService) FetchActualKlines(coin *domains.Coin, intervalInMinutes int) {
	lastKline, err := s.klineRepo.FindLast(coin.Id, fmt.Sprint(intervalInMinutes))
	if err != nil {
		zap.S().Errorf("Error FindLast %s", err.Error())
		return
	}
	var fetchKlinesFrom time.Time
	if lastKline == nil {
		fetchKlinesFrom = s.Clock.NowTime().Add(time.Minute * time.Duration(intervalInMinutes) * (bybit.BYBIT_MAX_LIMIT) * (-1))
	} else {
		fetchKlinesFrom = lastKline.OpenTime
		if s.Clock.NowTime().Before(lastKline.CloseTime) {
			return
		}
	}

	if err := s.FetchKlinesForPeriod(coin, fetchKlinesFrom, s.Clock.NowTime(), fmt.Sprint(intervalInMinutes)); err != nil {
		zap.S().Errorf("Error during fetchKlinesForPeriod %s", err.Error())
		return
	}
	return
}

func (s *KlinesFetcherService) FetchKlinesForPeriod(coin *domains.Coin, timeFrom time.Time, timeTo time.Time, interval string) error {
	timeIter := timeFrom
	for timeIter.Before(timeTo) {
		klinesDto, err := s.exchangeApi.GetKlines(coin, interval, bybit.BYBIT_MAX_LIMIT, timeIter)
		if err != nil {
			zap.S().Errorf("Error on fetch klines: %s", err)
			return err
		}
		if len(klinesDto.GetKlines()) == 0 {
			zap.S().Errorf("Empty response on fetch klines requestTime=%v %v",
				timeIter.Format(constants.DATE_TIME_FORMAT), klinesDto.String())
			return errors.New("Empty response on fetch klines.")
		}

		s.saveKlines(coin, klinesDto)

		klineLength := len(klinesDto.GetKlines())
		lastKline := klinesDto.GetKlines()[klineLength-1]
		timeIter = lastKline.GetCloseAt()
	}

	return nil
}

func (s *KlinesFetcherService) saveKlines(coin *domains.Coin, klinesDto api.KlinesDto) {
	for _, dto := range klinesDto.GetKlines() {
		existedKline, _ := s.klineRepo.FindOpenedAtMoment(coin.Id, dto.GetStartAt(), dto.GetInterval())

		if existedKline == nil {
			existedKline = &domains.Kline{
				CoinId:    coin.Id,
				OpenTime:  dto.GetStartAt(),
				CloseTime: dto.GetCloseAt(),
				Interval:  dto.GetInterval(),
				Open:      dto.GetOpen(),
			}
		}

		existedKline.High = dto.GetHigh()
		existedKline.Low = dto.GetLow()
		existedKline.Close = dto.GetClose()

		_ = s.klineRepo.SaveKline(existedKline)
	}
}
