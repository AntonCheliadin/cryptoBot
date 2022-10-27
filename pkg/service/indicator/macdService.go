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
		TechanConvertorService: techanConvertorService,
	}
	return macdServiceImpl
}

//Moving average convergence divergence
type MACDService struct {
	TechanConvertorService *techanLib.TechanConvertorService
}

func (s *MACDService) GetMacdHistogramIndicator(coin *domains.Coin, candleDuration string, fastLength int, slowLength int, signalLength int) *techan.Indicator {
	series := s.TechanConvertorService.BuildTimeSeriesByKlines(coin, candleDuration, int64(slowLength*4))

	macdIndicator := techan.NewMACDIndicator(techan.NewClosePriceIndicator(series), fastLength, slowLength)
	macdHistogramIndicator := techan.NewMACDHistogramIndicator(macdIndicator, signalLength)

	return &macdHistogramIndicator
}

func (s *MACDService) CalculateCurrentMACD(coin *domains.Coin, candleDuration string, fastLength int, slowLength int, signalLength int) big.Decimal {
	series := s.TechanConvertorService.BuildTimeSeriesByKlines(coin, candleDuration, int64(slowLength*4))

	macdIndicator := techan.NewMACDIndicator(techan.NewClosePriceIndicator(series), fastLength, slowLength)
	macdHistogramIndicator := techan.NewMACDHistogramIndicator(macdIndicator, signalLength)

	return macdHistogramIndicator.Calculate(len(series.Candles) - 1)
}

func (s *MACDService) CalculateMACDForAll(coin *domains.Coin, candleDuration string, fastLength int, slowLength int, signalLength int) []big.Decimal {
	series := s.TechanConvertorService.BuildTimeSeriesByKlines(coin, candleDuration, int64(slowLength*4))

	macdIndicator := techan.NewMACDIndicator(techan.NewClosePriceIndicator(series), fastLength, slowLength)
	macdHistogramIndicator := techan.NewMACDHistogramIndicator(macdIndicator, signalLength)

	resultList := make([]big.Decimal, len(series.Candles))

	for i := 0; i < len(series.Candles); i++ {
		resultList = append(resultList, macdHistogramIndicator.Calculate(i))
	}

	return resultList
}
