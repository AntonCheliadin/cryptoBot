package indicator

import (
	"cryptoBot/pkg/data/domains"
	"cryptoBot/pkg/repository"
	"cryptoBot/pkg/service/date"
	"cryptoBot/pkg/service/indicator/techanLib"
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
		techanConvertorService: techanConvertorService,
	}
	return standardDeviationServiceImpl
}

type StandardDeviationService struct {
	klineRepo              repository.Kline
	Clock                  date.Clock
	techanConvertorService *techanLib.TechanConvertorService
}

func (s *StandardDeviationService) calculateStandardDeviation(coin *domains.Coin, candleDuration string) big.Decimal {
	series := s.techanConvertorService.BuildTimeSeriesByKlines(coin, candleDuration, viper.GetInt64("indicator.standardDeviation.length")+1)

	stdDev := techan.NewStandardDeviationIndicator(techan.NewClosePriceIndicator(series))

	return stdDev.Calculate(viper.GetInt("indicator.standardDeviation.length"))
}

func (s *StandardDeviationService) IsVolatilityOscillatorSignal(coin *domains.Coin, candleDuration string) bool {
	stdDev := s.calculateStandardDeviation(coin, candleDuration)

	kline, _ := s.klineRepo.FindOpenedAtMoment(coin.Id, s.Clock.NowTime(), candleDuration)

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
