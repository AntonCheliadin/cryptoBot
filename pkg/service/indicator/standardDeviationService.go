package indicator

import (
	"cryptoBot/pkg/data/domains"
	"cryptoBot/pkg/repository"
	"cryptoBot/pkg/service/date"
	"github.com/sdcoffey/big"
	"github.com/sdcoffey/techan"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"time"
)

var standardDeviationServiceImpl *StandardDeviationService

func NewStandardDeviationService(clock date.Clock, klineRepo repository.Kline) *StandardDeviationService {
	if standardDeviationServiceImpl != nil {
		panic("Unexpected try to create second service instance")
	}
	standardDeviationServiceImpl = &StandardDeviationService{
		klineRepo: klineRepo,
		Clock:     clock,
	}
	return standardDeviationServiceImpl
}

type StandardDeviationService struct {
	klineRepo repository.Kline
	Clock     date.Clock
}

func (s *StandardDeviationService) calculateStandardDeviation(coin *domains.Coin, candleDuration string) big.Decimal {
	series := techan.NewTimeSeries()

	klineSizeForStandardDeviationLength := viper.GetInt64("indicator.standardDeviation.length") + 1
	klines, err := s.klineRepo.FindAllByCoinIdAndIntervalAndCloseTimeLessOrderByOpenTimeWithLimit(coin.Id, candleDuration, s.Clock.NowTime(), klineSizeForStandardDeviationLength)
	if err != nil {
		zap.S().Errorf("Error on FindAllByCoinIdAndIntervalAndCloseTimeLessOrderByOpenTimeWithLimit: %s", err)
		return big.NaN
	}

	// fetch this from your preferred exchange
	for _, kline := range klines {
		period := techan.NewTimePeriod(kline.OpenTime, time.Minute*time.Duration(kline.GetIntervalInMinutes()))

		candle := techan.NewCandle(period)
		candle.OpenPrice = big.NewDecimal(float64(kline.Open) / 100)
		candle.ClosePrice = big.NewDecimal(float64(kline.Close) / 100)
		candle.MaxPrice = big.NewDecimal(float64(kline.High) / 100)
		candle.MinPrice = big.NewDecimal(float64(kline.Low) / 100)

		series.AddCandle(candle)
	}

	stdDev := techan.NewStandardDeviationIndicator(techan.NewClosePriceIndicator(series))

	return stdDev.Calculate(viper.GetInt("indicator.standardDeviation.length"))
}

func (s *StandardDeviationService) IsVolatilityOscillatorSignal(coin *domains.Coin, candleDuration string) bool {
	stdDev := s.calculateStandardDeviation(coin, candleDuration)

	kline, _ := s.klineRepo.FindClosedAtMoment(coin.Id, s.Clock.NowTime(), candleDuration)

	priceChange := kline.GetPriceChange()
	priceChangeInDecimal := big.NewDecimal(float64(priceChange) / 100)

	stdDeviationPercent := big.NewDecimal(viper.GetFloat64("indicator.standardDeviation.percent"))
	partOfStdDev := stdDev.Mul(stdDeviationPercent)

	zap.S().Infof("priceChange=[%v] stdDev=[%v] partOfStdDev=[%v]",
		priceChangeInDecimal.FormattedString(2), stdDev.FormattedString(2), partOfStdDev.FormattedString(2))

	if priceChange > 0 {
		return priceChangeInDecimal.GTE(partOfStdDev)
	} else {
		return priceChangeInDecimal.LTE(partOfStdDev.Mul(big.NewFromInt(-1)))
	}
}
