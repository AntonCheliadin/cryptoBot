package indicator

import (
	"cryptoBot/pkg/data/domains"
	"cryptoBot/pkg/repository"
	"cryptoBot/pkg/service/date"
	"cryptoBot/pkg/util"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

var movingAverageServiceImpl *MovingAverageService

func NewMovingAverageService(clock date.Clock, klineRepo repository.Kline) *MovingAverageService {
	if movingAverageServiceImpl != nil {
		panic("Unexpected try to create second service instance")
	}
	movingAverageServiceImpl = &MovingAverageService{
		klineRepo: klineRepo,
		Clock:     clock,
	}
	return movingAverageServiceImpl
}

type MovingAverageService struct {
	klineRepo repository.Kline
	Clock     date.Clock
}

/**
Return two last points of moving averages
*/
func (s *MovingAverageService) CalculateAvg(coin *domains.Coin, length int, returnPointsSize int) []float64 {
	candleDuration := viper.GetString("strategy.ma.interval")
	klines, err := s.klineRepo.FindAllByCoinIdAndIntervalAndCloseTimeLessOrderByOpenTimeWithLimit(coin.Id, candleDuration, s.Clock.NowTime(), int64(length+returnPointsSize-1))
	if err != nil {
		zap.S().Errorf("Error on FindAllByCoinIdAndIntervalAndCloseTimeLessOrderByOpenTimeWithLimit: %s", err)
		return nil
	}

	var avgPoints []float64
	var movingAvgPoints []float64

	for _, kline := range klines {
		avgPoints = append(avgPoints, kline.Close /*(kline.Open+kline.Close+kline.High+kline.Low)/4*/)

		if len(avgPoints) == length {
			averageByLength := util.SumFloat64(avgPoints) / float64(length)
			movingAvgPoints = append(movingAvgPoints, averageByLength)

			avgPoints = avgPoints[1:] //remove first element
		}
	}

	if movingAvgPoints == nil || len(movingAvgPoints) < returnPointsSize {
		zap.S().Errorf("Can't calculate enough points [%v] of moving averages", returnPointsSize)
		return nil
	}

	return movingAvgPoints
}
