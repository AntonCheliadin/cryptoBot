package indicator

import (
	"cryptoBot/pkg/constants"
	"cryptoBot/pkg/constants/futureType"
	"cryptoBot/pkg/data/domains"
	"cryptoBot/pkg/repository"
	"cryptoBot/pkg/service/date"
	"github.com/sdcoffey/techan"
	"go.uber.org/zap"
	"strconv"
	"time"
)

var localExtremumTrendServiceImpl *LocalExtremumTrendService

func NewLocalExtremumTrendService(clock date.Clock, klineRepo repository.Kline) *LocalExtremumTrendService {
	if localExtremumTrendServiceImpl != nil {
		panic("Unexpected try to create second service instance")
	}
	localExtremumTrendServiceImpl = &LocalExtremumTrendService{
		klineRepo: klineRepo,
		Clock:     clock,
	}
	return localExtremumTrendServiceImpl
}

type LocalExtremumTrendService struct {
	klineRepo repository.Kline
	Clock     date.Clock
}

func (s *LocalExtremumTrendService) IsTrendUp(coin *domains.Coin, klinesInterval string) bool {
	nextLowKline := s.findNearestLowExtremum(coin, klinesInterval, s.Clock.NowTime())
	nextHighKline := s.findNearestHighExtremum(coin, klinesInterval, nextLowKline.CloseTime)

	prevLowKline := s.findNearestLowExtremum(coin, klinesInterval, nextHighKline.CloseTime)
	prevHighKline := s.findNearestHighExtremum(coin, klinesInterval, prevLowKline.CloseTime)

	zap.S().Infof("nextLowKline %v at [%v - %v]", nextLowKline.High, nextLowKline.OpenTime.Format(constants.DATE_TIME_FORMAT), nextLowKline.CloseTime.Format(constants.DATE_TIME_FORMAT))
	zap.S().Infof("nextHighKline %v at [%v - %v]", nextHighKline.High, nextHighKline.OpenTime.Format(constants.DATE_TIME_FORMAT), nextHighKline.CloseTime.Format(constants.DATE_TIME_FORMAT))
	zap.S().Infof("prevLowKline %v at [%v - %v]", prevLowKline.High, prevLowKline.OpenTime.Format(constants.DATE_TIME_FORMAT), prevLowKline.CloseTime.Format(constants.DATE_TIME_FORMAT))
	zap.S().Infof("prevHighKline %v at [%v - %v]", prevHighKline.High, prevHighKline.OpenTime.Format(constants.DATE_TIME_FORMAT), prevHighKline.CloseTime.Format(constants.DATE_TIME_FORMAT))

	isHigherHighAndHigherLow := nextHighKline.High > prevHighKline.High && nextLowKline.Low > prevLowKline.Low

	zap.S().Infof("IsTrendUp isHigherHighAndHigherLow = %v", isHigherHighAndHigherLow)

	return isHigherHighAndHigherLow
}

func (s *LocalExtremumTrendService) IsTrendDown(coin *domains.Coin, klinesInterval string) bool {
	nextHighKline := s.findNearestHighExtremum(coin, klinesInterval, s.Clock.NowTime())
	nextLowKline := s.findNearestLowExtremum(coin, klinesInterval, nextHighKline.CloseTime)

	prevHighKline := s.findNearestHighExtremum(coin, klinesInterval, nextLowKline.CloseTime)
	prevLowKline := s.findNearestLowExtremum(coin, klinesInterval, prevHighKline.CloseTime)

	zap.S().Infof("nextHighKline %v at [%v - %v]", nextHighKline.High, nextHighKline.OpenTime.Format(constants.DATE_TIME_FORMAT), nextHighKline.CloseTime.Format(constants.DATE_TIME_FORMAT))
	zap.S().Infof("nextLowKline %v at [%v - %v]", nextLowKline.High, nextLowKline.OpenTime.Format(constants.DATE_TIME_FORMAT), nextLowKline.CloseTime.Format(constants.DATE_TIME_FORMAT))
	zap.S().Infof("prevHighKline %v at [%v - %v]", prevHighKline.High, prevHighKline.OpenTime.Format(constants.DATE_TIME_FORMAT), prevHighKline.CloseTime.Format(constants.DATE_TIME_FORMAT))
	zap.S().Infof("prevLowKline %v at [%v - %v]", prevLowKline.High, prevLowKline.OpenTime.Format(constants.DATE_TIME_FORMAT), prevLowKline.CloseTime.Format(constants.DATE_TIME_FORMAT))

	isLowerLowAndLowerHigh := prevHighKline.High > nextHighKline.High && prevLowKline.Low > nextLowKline.Low

	zap.S().Infof("IsTrendDown isLowerLowAndLowerHigh = %v", isLowerLowAndLowerHigh)

	return isLowerLowAndLowerHigh
}

func (s *LocalExtremumTrendService) CalculateStopLoss(coin *domains.Coin, klinesInterval string, futuresType futureType.FuturesType) int64 {
	if futuresType == futureType.SHORT {
		highExtremumKline := s.findNearestHighExtremum(coin, klinesInterval, s.Clock.NowTime())
		return highExtremumKline.High + int64(float64(highExtremumKline.High)*0.0003) // +0.03%
	} else {
		lowExtremumKline := s.findNearestLowExtremum(coin, klinesInterval, s.Clock.NowTime())
		return lowExtremumKline.Low - int64(float64(lowExtremumKline.Low)*0.0003) // -0.03%
	}
}

func (s *LocalExtremumTrendService) findNearestHighExtremum(coin *domains.Coin, klinesInterval string, timeIter time.Time) *domains.Kline {
	var highExtremumKline *domains.Kline
	minExtremumWindow := 2

	for extremumWindowCounter := 0; extremumWindowCounter < minExtremumWindow; timeIter = timeIter.Add(time.Minute * -1) {
		kline, _ := s.klineRepo.FindClosedAtMoment(coin.Id, timeIter, klinesInterval)

		if highExtremumKline == nil || kline.High > highExtremumKline.High {
			highExtremumKline = kline
			extremumWindowCounter = 0
		} else {
			extremumWindowCounter++
		}
	}

	zap.S().Infof("findNearestHighExtremum %v    [%v]", highExtremumKline.High, timeIter.Format(constants.DATE_TIME_FORMAT))

	return highExtremumKline
}

func (s *LocalExtremumTrendService) findNearestLowExtremum(coin *domains.Coin, klinesInterval string, timeIter time.Time) *domains.Kline {
	var lowExtremumKline *domains.Kline
	minExtremumWindow := 2

	klinesIntervalInt, _ := strconv.Atoi(klinesInterval)
	for extremumWindowCounter := 0; extremumWindowCounter < minExtremumWindow; timeIter = timeIter.Add(time.Minute * time.Duration(klinesIntervalInt) * -1) {
		kline, _ := s.klineRepo.FindClosedAtMoment(coin.Id, timeIter, klinesInterval)

		if lowExtremumKline == nil || kline.Low < lowExtremumKline.Low {
			lowExtremumKline = kline
			extremumWindowCounter = 0
		} else {
			extremumWindowCounter++
		}
	}

	zap.S().Infof("findNearestLowExtremum %v    [%v]", lowExtremumKline.Low, timeIter.Format(constants.DATE_TIME_FORMAT))

	return lowExtremumKline
}

//delete below

func (s *LocalExtremumTrendService) CrossTheFastSmaByTrendSignal(series *techan.TimeSeries, fastSMA, slowSMA techan.Indicator) (bool, futureType.FuturesType) {
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

func (s *LocalExtremumTrendService) isTheSameTrendForLength(series *techan.TimeSeries, fastSMA, slowSMA techan.Indicator, minTubeLength int) bool {
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

func (s *LocalExtremumTrendService) IsLastKlineClosedInTube(series *techan.TimeSeries, fastSMA, slowSMA techan.Indicator) bool {
	lastIndex := len(series.Candles) - 1

	isFastBelow := fastSMA.Calculate(lastIndex).LT(slowSMA.Calculate(lastIndex))
	if isFastBelow {
		return s.IsKlineClosedInTube(series, lastIndex, fastSMA, slowSMA)
	} else {
		return s.IsKlineClosedInTube(series, lastIndex, slowSMA, fastSMA)
	}
}

func (s *LocalExtremumTrendService) HasLastKlineGotOutFromTube(series *techan.TimeSeries, fastSMA, slowSMA techan.Indicator) bool {
	lastIndex := len(series.Candles) - 1
	candle := series.Candles[lastIndex]

	isFastBelow := fastSMA.Calculate(lastIndex).LT(slowSMA.Calculate(lastIndex))
	if isFastBelow {
		return candle.OpenPrice.LT(slowSMA.Calculate(lastIndex)) && candle.ClosePrice.GT(slowSMA.Calculate(lastIndex))
	} else {
		return candle.OpenPrice.GT(slowSMA.Calculate(lastIndex)) && candle.ClosePrice.LT(slowSMA.Calculate(lastIndex))
	}
}

func (s *LocalExtremumTrendService) IsKlineClosedInTube(series *techan.TimeSeries, candleIndex int, lowSMA, highSMA techan.Indicator) bool {
	candle := series.Candles[candleIndex]

	smaHighVal := highSMA.Calculate(candleIndex)
	smaLowVal := lowSMA.Calculate(candleIndex)

	isInTube := candle.ClosePrice.LTE(smaHighVal) && candle.ClosePrice.GTE(smaLowVal)

	return isInTube
}
