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
	return s.BuildTimeSeriesByKlinesAtMoment(coin, candleDuration, length, s.Clock.NowTime())
}

func (s *TechanConvertorService) BuildTimeSeriesByKlinesAtMoment(coin *domains.Coin, candleDuration string, length int64, moment time.Time) *techan.TimeSeries {
	series := techan.NewTimeSeries()

	klines, err := s.klineRepo.FindAllByCoinIdAndIntervalAndCloseTimeLessOrderByOpenTimeWithLimit(coin.Id, candleDuration, moment, length)
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
		candle.Volume = big.NewDecimal(kline.Volume)

		series.AddCandle(candle)
	}

	return series
}

func (s *TechanConvertorService) ConvertKlinesToSeries(klines []domains.IKline, candleDuration int) *techan.TimeSeries {
	series := techan.NewTimeSeries()

	for _, kline := range klines {
		period := techan.NewTimePeriod(kline.GetOpenTime(), time.Minute*time.Duration(candleDuration))

		candle := techan.NewCandle(period)
		candle.OpenPrice = big.NewDecimal(float64(kline.GetOpen()) / 100)
		candle.ClosePrice = big.NewDecimal(float64(kline.GetClose()) / 100)

		series.AddCandle(candle)
	}

	return series
}
