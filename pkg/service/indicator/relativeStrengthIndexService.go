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
		techanConvertorService: techanConvertorService,
	}
	return relativeStrengthIndexServiceImpl
}

type RelativeStrengthIndexService struct {
	techanConvertorService *techanLib.TechanConvertorService
}

func (s *RelativeStrengthIndexService) CalculateRSI(coin *domains.Coin, candleDuration string, length int) big.Decimal {
	series := s.techanConvertorService.BuildTimeSeriesByKlines(coin, candleDuration, int64(length*4))

	rsiIndicator := techan.NewRelativeStrengthIndexIndicator(techan.NewClosePriceIndicator(series), length)

	return rsiIndicator.Calculate(len(series.Candles) - 1)
}
