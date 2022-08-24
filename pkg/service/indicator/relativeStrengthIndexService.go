package indicator

import (
	"cryptoBot/pkg/data/domains"
	"cryptoBot/pkg/service/indicator/techanLib"
	"github.com/sdcoffey/big"
	"github.com/sdcoffey/techan"
	"go.uber.org/zap"
)

var relativeStrengthIndexServiceImpl *RelativeStrengthIndexService

func NewRelativeStrengthIndexService(techanConvertorService *techanLib.TechanConvertorService) *RelativeStrengthIndexService {
	if relativeStrengthIndexServiceImpl != nil {
		panic("Unexpected try to create second service instance")
	}
	relativeStrengthIndexServiceImpl = &RelativeStrengthIndexService{
		techanConvertorService: techanConvertorService,
	}
	return relativeStrengthIndexServiceImpl
}

type RelativeStrengthIndexService struct {
	techanConvertorService *techanLib.TechanConvertorService
}

func (s *RelativeStrengthIndexService) CalculateRSI(coin *domains.Coin, candleDuration string, length int) big.Decimal {
	series := s.techanConvertorService.BuildTimeSeriesByKlines(coin, candleDuration, int64(length*4))
	//candleDurationInt, _ := strconv.Atoi(candleDuration)

	rsiIndicator := techan.NewRelativeStrengthIndexIndicator(techan.NewClosePriceIndicator(series), length)

	rsi := make([]string, len(series.Candles))

	for i := 0; i < len(series.Candles); i++ {
		rsi[i] = rsiIndicator.Calculate(i).String()
		zap.S().Infof("RSI=%v for candle %v", rsi[i], series.Candles[i].String())
	}
	zap.S().Infof("RSI=%v for %v timeFrame", rsi, length)

	return rsiIndicator.Calculate(len(series.Candles) - 1)
}
