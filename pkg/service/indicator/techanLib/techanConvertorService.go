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
		candle.OpenPrice = big.NewDecimal(kline.Open)
		candle.ClosePrice = big.NewDecimal(kline.Close)
		candle.MaxPrice = big.NewDecimal(kline.High)
		candle.MinPrice = big.NewDecimal(kline.Low)
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
		candle.OpenPrice = big.NewDecimal(kline.GetOpen())
		candle.ClosePrice = big.NewDecimal(kline.GetClose())

		series.AddCandle(candle)
	}

	return series
}
