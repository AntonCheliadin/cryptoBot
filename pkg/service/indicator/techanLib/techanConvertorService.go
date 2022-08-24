package techanLib

import (
	"cryptoBot/pkg/data/domains"
	"cryptoBot/pkg/repository"
	"cryptoBot/pkg/service/date"
	"github.com/sdcoffey/big"
	"github.com/sdcoffey/techan"
	"go.uber.org/zap"
	"strconv"
	"time"
)

var techanConvertorServiceImpl *TechanConvertorService

func NewTechanConvertorService(clock date.Clock, klineRepo repository.Kline) *TechanConvertorService {
	if techanConvertorServiceImpl != nil {
		panic("Unexpected try to create second service instance")
	}
	techanConvertorServiceImpl = &TechanConvertorService{
		klineRepo: klineRepo,
		Clock:     clock,
	}
	return techanConvertorServiceImpl
}

type TechanConvertorService struct {
	klineRepo repository.Kline
	Clock     date.Clock
}

func (s *TechanConvertorService) BuildTimeSeriesByKlines(coin *domains.Coin, candleDuration string, length int64) *techan.TimeSeries {
	series := techan.NewTimeSeries()

	klines, err := s.klineRepo.FindAllByCoinIdAndIntervalAndCloseTimeLessOrderByOpenTimeWithLimit(coin.Id, candleDuration, s.Clock.NowTime(), length)
	if err != nil {
		zap.S().Errorf("Error on FindAllByCoinIdAndIntervalAndCloseTimeLessOrderByOpenTimeWithLimit: %s", err)
		return nil
	}

	candleDurationInt, _ := strconv.Atoi(candleDuration)

	for _, kline := range klines {
		period := techan.NewTimePeriod(kline.OpenTime, time.Minute*time.Duration(candleDurationInt))

		candle := techan.NewCandle(period)
		candle.OpenPrice = big.NewDecimal(float64(kline.Open) / 100)
		candle.ClosePrice = big.NewDecimal(float64(kline.Close) / 100)
		candle.MaxPrice = big.NewDecimal(float64(kline.High) / 100)
		candle.MinPrice = big.NewDecimal(float64(kline.Low) / 100)

		series.AddCandle(candle)
	}

	return series
}
