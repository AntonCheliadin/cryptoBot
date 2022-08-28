package indicator

import (
	"cryptoBot/pkg/data/domains"
	"cryptoBot/pkg/service/indicator/techanLib"
	"github.com/sdcoffey/big"
	"github.com/sdcoffey/techan"
)

var macdServiceImpl *MACDService

func NewMACDService(techanConvertorService *techanLib.TechanConvertorService) *MACDService {
	if relativeStrengthIndexServiceImpl != nil {
		panic("Unexpected try to create second service instance")
	}
	macdServiceImpl = &MACDService{
		techanConvertorService: techanConvertorService,
	}
	return macdServiceImpl
}

//Moving average convergence divergence
type MACDService struct {
	techanConvertorService *techanLib.TechanConvertorService
}

func (s *MACDService) CalculateMACD(coin *domains.Coin, candleDuration string, fastLength int, slowLength int, signalLength int) big.Decimal {
	series := s.techanConvertorService.BuildTimeSeriesByKlines(coin, candleDuration, int64(slowLength*4))

	macdIndicator := techan.NewMACDIndicator(techan.NewClosePriceIndicator(series), fastLength, slowLength)
	macdHistogramIndicator := techan.NewMACDHistogramIndicator(macdIndicator, signalLength)

	return macdHistogramIndicator.Calculate(len(series.Candles) - 1)
}
