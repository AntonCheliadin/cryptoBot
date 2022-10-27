package indicator

import (
	"cryptoBot/pkg/constants/futureType"
	"cryptoBot/pkg/data/domains"
	"cryptoBot/pkg/repository"
	"cryptoBot/pkg/service/date"
	"cryptoBot/pkg/service/indicator/techanLib"
	"cryptoBot/pkg/util"
	"github.com/sdcoffey/big"
	"github.com/sdcoffey/techan"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

var standardDeviationServiceImpl *StandardDeviationService

func NewStandardDeviationService(clock date.Clock, klineRepo repository.Kline, techanConvertorService *techanLib.TechanConvertorService) *StandardDeviationService {
	if standardDeviationServiceImpl != nil {
		panic("Unexpected try to create second service instance")
	}
	standardDeviationServiceImpl = &StandardDeviationService{
		klineRepo:              klineRepo,
		Clock:                  clock,
		TechanConvertorService: techanConvertorService,
	}
	return standardDeviationServiceImpl
}

type StandardDeviationService struct {
	klineRepo              repository.Kline
	Clock                  date.Clock
	TechanConvertorService *techanLib.TechanConvertorService
}

func (s *StandardDeviationService) CalculateCurrentStandardDeviation(coin *domains.Coin, candleDuration string) big.Decimal {
	series := s.TechanConvertorService.BuildTimeSeriesByKlines(coin, candleDuration, viper.GetInt64("indicator.standardDeviation.length")+1)

	stdDev := techan.NewStandardDeviationIndicator(techan.NewClosePriceIndicator(series))

	return stdDev.Calculate(viper.GetInt("indicator.standardDeviation.length"))
}

func (s *StandardDeviationService) CalculateCurrentStandardDeviationForPriceChange(coin *domains.Coin, candleDuration string) big.Decimal {
	klines, err := s.klineRepo.FindAllByCoinIdAndIntervalAndCloseTimeLessOrderByOpenTimeWithLimit(coin.Id, candleDuration, s.Clock.NowTime(), viper.GetInt64("indicator.standardDeviation.length"))
	if err != nil {
		zap.S().Errorf("Error on FindAllByCoinIdAndIntervalAndCloseTimeLessOrderByOpenTimeWithLimit: %s", err)
		return big.ZERO
	}

	priceChangeArray := make([]float64, len(klines))

	for i, kline := range klines {
		priceChangeArray[i] = float64(kline.Close) - float64(kline.Open)
	}

	return big.NewDecimal(util.StandardDeviation(priceChangeArray))
}

func (s *StandardDeviationService) IsVolatilityOscillatorSignal(coin *domains.Coin, candleDuration string) (bool, futureType.FuturesType) {
	stdDev := s.CalculateCurrentStandardDeviationForPriceChange(coin, candleDuration)

	kline, err := s.klineRepo.FindClosedAtMoment(coin.Id, s.Clock.NowTime(), candleDuration)
	if err != nil || kline == nil {
		zap.S().Errorf("Failed to FindClosedAtMoment %v - %s", s.Clock.NowTime(), err)
		return false, futureType.SHORT
	}

	priceChange := kline.GetPriceChange()
	priceChangeInDecimal := big.NewDecimal(float64(priceChange))

	stdDeviationPercent := big.NewDecimal(viper.GetFloat64("indicator.standardDeviation.percent"))
	partOfStdDev := stdDev.Mul(stdDeviationPercent)

	zap.S().Debugf("priceChange=[%v] stdDev=[%v] partOfStdDev=[%v]",
		priceChangeInDecimal.FormattedString(2), stdDev.FormattedString(2), partOfStdDev.FormattedString(2))

	if priceChange > 0 {
		return priceChangeInDecimal.GTE(partOfStdDev), futureType.LONG
	} else {
		return priceChangeInDecimal.LTE(partOfStdDev.Mul(big.NewFromInt(-1))), futureType.SHORT
	}
}
