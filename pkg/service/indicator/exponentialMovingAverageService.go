package indicator

import (
	"cryptoBot/pkg/constants/indicator"
	"cryptoBot/pkg/data/domains"
	"cryptoBot/pkg/service/indicator/techanLib"
	"github.com/sdcoffey/big"
	"github.com/sdcoffey/techan"
)

var exponentialMovingAverageServiceImpl *ExponentialMovingAverageService

func NewExponentialMovingAverageService(techanConvertorService *techanLib.TechanConvertorService) *ExponentialMovingAverageService {
	if exponentialMovingAverageServiceImpl != nil {
		panic("Unexpected try to create second service instance")
	}
	exponentialMovingAverageServiceImpl = &ExponentialMovingAverageService{
		techanConvertorService: techanConvertorService,
	}
	return exponentialMovingAverageServiceImpl
}

type ExponentialMovingAverageService struct {
	techanConvertorService *techanLib.TechanConvertorService
}

func (s *ExponentialMovingAverageService) CalculateEMA(coin *domains.Coin, candleDuration string, length int) big.Decimal {
	series := s.techanConvertorService.BuildTimeSeriesByKlines(coin, candleDuration, int64(length))

	emaIndicator := techan.NewEMAIndicator(techan.NewClosePriceIndicator(series), length)

	return emaIndicator.Calculate(len(series.Candles) - 1)
}

func (s *ExponentialMovingAverageService) IsFastEmaAbove(coin *domains.Coin, candleDuration string,
	fastLength int, fastType indicator.MovingAveragesType, slowLength int, slowType indicator.MovingAveragesType) bool {
	series := s.techanConvertorService.BuildTimeSeriesByKlines(coin, candleDuration, int64(slowLength))
	closePriceIndicator := techan.NewClosePriceIndicator(series)

	fastIndicator := s.buildIndicator(closePriceIndicator, fastLength, fastType)
	slowIndicator := s.buildIndicator(closePriceIndicator, slowLength, slowType)

	fastEMA := fastIndicator.Calculate(len(series.Candles) - 1)
	slowEMA := slowIndicator.Calculate(len(series.Candles) - 1)

	return fastEMA.GT(slowEMA)
}

func (s *ExponentialMovingAverageService) buildIndicator(closePriceIndicator techan.Indicator, length int, maType indicator.MovingAveragesType) techan.Indicator {
	if maType == indicator.EMA {
		return techan.NewEMAIndicator(closePriceIndicator, length)
	} else if maType == indicator.SMA {
		return techan.NewSimpleMovingAverage(closePriceIndicator, length)
	} else {
		return nil
	}
}
