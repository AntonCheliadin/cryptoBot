package indicator

import (
	"cryptoBot/pkg/constants/futureType"
	"cryptoBot/pkg/repository"
	"cryptoBot/pkg/service/date"
	"github.com/sdcoffey/techan"
)

var smaTubeServiceImpl *SmaTubeService

func NewSmaTubeService(clock date.Clock, klineRepo repository.Kline) *SmaTubeService {
	if smaTubeServiceImpl != nil {
		panic("Unexpected try to create second service instance")
	}
	smaTubeServiceImpl = &SmaTubeService{
		klineRepo: klineRepo,
		Clock:     clock,
	}
	return smaTubeServiceImpl
}

type SmaTubeService struct {
	klineRepo repository.Kline
	Clock     date.Clock
}

func (s *SmaTubeService) CrossTheFastSmaByTrendSignal(series *techan.TimeSeries, fastSMA, slowSMA techan.Indicator) (bool, futureType.FuturesType) {
	lastKlineIndex := len(series.Candles) - 1
	candle := series.Candles[lastKlineIndex]
	fastSmaLastValue := fastSMA.Calculate(lastKlineIndex)
	slowSmaLastValue := slowSMA.Calculate(lastKlineIndex)

	isFastBelow := fastSmaLastValue.LT(slowSmaLastValue)
	if isFastBelow {
		return candle.OpenPrice.GTE(fastSmaLastValue) && candle.ClosePrice.LTE(fastSmaLastValue), futureType.SHORT
	} else {
		return candle.OpenPrice.LTE(fastSmaLastValue) && candle.ClosePrice.GTE(fastSmaLastValue), futureType.LONG
	}
}

func (s *SmaTubeService) isTheSameTrendForLength(series *techan.TimeSeries, fastSMA, slowSMA techan.Indicator, minTubeLength int) bool {
	isFastAbove := true

	for i := 1; i < minTubeLength; i++ {
		candleIndex := len(series.Candles) - i

		fastSMA := fastSMA.Calculate(candleIndex)
		slowSMA := slowSMA.Calculate(candleIndex)

		isCurrentFastAbove := fastSMA.GT(slowSMA)

		if i == 1 {
			isFastAbove = isCurrentFastAbove
		} else if isCurrentFastAbove != isFastAbove {
			return false
		}
	}

	return true
}

func (s *SmaTubeService) IsLastKlineClosedInTube(series *techan.TimeSeries, fastSMA, slowSMA techan.Indicator) bool {
	lastIndex := len(series.Candles) - 1

	isFastBelow := fastSMA.Calculate(lastIndex).LT(slowSMA.Calculate(lastIndex))
	if isFastBelow {
		return s.IsKlineClosedInTube(series, lastIndex, fastSMA, slowSMA)
	} else {
		return s.IsKlineClosedInTube(series, lastIndex, slowSMA, fastSMA)
	}
}

func (s *SmaTubeService) HasLastKlineGotOutFromTube(series *techan.TimeSeries, fastSMA, slowSMA techan.Indicator) bool {
	lastIndex := len(series.Candles) - 1
	candle := series.Candles[lastIndex]

	isFastBelow := fastSMA.Calculate(lastIndex).LT(slowSMA.Calculate(lastIndex))
	if isFastBelow {
		return candle.OpenPrice.LT(slowSMA.Calculate(lastIndex)) && candle.ClosePrice.GT(slowSMA.Calculate(lastIndex))
	} else {
		return candle.OpenPrice.GT(slowSMA.Calculate(lastIndex)) && candle.ClosePrice.LT(slowSMA.Calculate(lastIndex))
	}
}

func (s *SmaTubeService) IsKlineClosedInTube(series *techan.TimeSeries, candleIndex int, lowSMA, highSMA techan.Indicator) bool {
	candle := series.Candles[candleIndex]

	smaHighVal := highSMA.Calculate(candleIndex)
	smaLowVal := lowSMA.Calculate(candleIndex)

	isInTube := candle.ClosePrice.LTE(smaHighVal) && candle.ClosePrice.GTE(smaLowVal)

	return isInTube
}
