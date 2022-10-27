package exchange

import (
	"cryptoBot/pkg/api"
	"cryptoBot/pkg/constants"
	"cryptoBot/pkg/constants/bybit"
	"cryptoBot/pkg/data/domains"
	"cryptoBot/pkg/repository"
	"errors"
	"go.uber.org/zap"
	"time"
)

var klinesFetcherServiceImpl *KlinesFetcherService

func NewKlinesFetcherService(exchangeApi api.ExchangeApi, klineRepo repository.Kline) *KlinesFetcherService {
	if klinesFetcherServiceImpl != nil {
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
