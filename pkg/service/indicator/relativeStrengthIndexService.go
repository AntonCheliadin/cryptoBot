package indicator

import (
	"cryptoBot/pkg/data/domains"
	"cryptoBot/pkg/service/indicator/techanLib"
	"github.com/sdcoffey/big"
	"github.com/sdcoffey/techan"
)

var relativeStrengthIndexServiceImpl *RelativeStrengthIndexService

func NewRelativeStrengthIndexService(techanConvertorService *techanLib.TechanConvertorService) *RelativeStrengthIndexService {
	if relativeStrengthIndexServiceImpl != nil {
		panic("Unexpected try to create second service instance")
	}
	relativeStrengthIndexServiceImpl = &RelativeStrengthIndexService{
		TechanConvertorService: techanConvertorService,
	}
	return relativeStrengthIndexServiceImpl
}

type RelativeStrengthIndexService struct {
	TechanConvertorService *techanLib.TechanConvertorService
}

func (s *RelativeStrengthIndexService) CalculateCurrentRSI(coin *domains.Coin, candleDuration string, length int) big.Decimal {
	series := s.TechanConvertorService.BuildTimeSeriesByKlines(coin, candleDuration, int64(length*4))

	rsiIndicator := techan.NewRelativeStrengthIndexIndicator(techan.NewClosePriceIndicator(series), length)

	return rsiIndicator.Calculate(len(series.Candles) - 1)
}

func (s *RelativeStrengthIndexService) CalculateRSIForAll(coin *domains.Coin, candleDuration string, rsiLength int) []big.Decimal {
	series := s.TechanConvertorService.BuildTimeSeriesByKlines(coin, candleDuration, int64(rsiLength*4))

	rsiIndicator := techan.NewRelativeStrengthIndexIndicator(techan.NewClosePriceIndicator(series), rsiLength)

	resultList := make([]big.Decimal, len(series.Candles))

	for i := 0; i < len(series.Candles); i++ {
		resultList = append(resultList, rsiIndicator.Calculate(i))
	}

	return resultList
}
