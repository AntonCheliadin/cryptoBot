package orders

import (
	"cryptoBot/pkg/constants/futureType"
	"cryptoBot/pkg/data/domains"
	"cryptoBot/pkg/repository"
	"cryptoBot/pkg/service/date"
	"cryptoBot/pkg/util"
	"github.com/spf13/viper"
	"math"
	"time"
)

var profitLossFinderServiceImpl *ProfitLossFinderService

func NewProfitLossFinderService(clock date.Clock, klineRepo repository.Kline) *ProfitLossFinderService {
	if profitLossFinderServiceImpl != nil {
		panic("Unexpected try to create second service instance")
	}
	profitLossFinderServiceImpl = &ProfitLossFinderService{
		klineRepo: klineRepo,
		Clock:     clock,
	}
	return profitLossFinderServiceImpl
}

type ProfitLossFinderService struct {
	klineRepo repository.Kline
	Clock     date.Clock
}

func (s *ProfitLossFinderService) FindStopLoss(coin *domains.Coin, time time.Time, klineInterval string, futuresType futureType.FuturesType) (int64, error) {
	klines, err := s.klineRepo.FindAllByCoinIdAndIntervalAndCloseTimeLessOrderByOpenTimeWithLimit(coin.Id, klineInterval, time, viper.GetInt64("orders.dynamicStopLoss.klinesLimit"))
	if err != nil {
		return 0, err
	}

	maxHigh := int64(0)
	minLow := int64(9223372036854775807)

	for _, kline := range klines {
		maxHigh = util.Max(maxHigh, kline.High)
		minLow = util.Min(minLow, kline.Low)
	}

	currentPrice := klines[len(klines)-1].Close

	return s.GetStopLossInConfigRange(currentPrice, minLow, maxHigh, futuresType), nil
}

func (s *ProfitLossFinderService) GetStopLossInConfigRange(currentPrice int64, minLow int64, maxHigh int64, futuresType futureType.FuturesType) int64 {
	localExtremum := maxHigh
	futuresTypeSign := int64(1)
	if futuresType == futureType.LONG {
		futuresTypeSign = -1
		localExtremum = minLow
	}

	maxHighInPercent := math.Abs(util.CalculateChangeInPercents(currentPrice, localExtremum))

	if maxHighInPercent > viper.GetFloat64("orders.dynamicStopLoss.maxPercent") {
		return currentPrice + int64(util.CalculatePercentOf(float64(currentPrice), viper.GetFloat64("orders.dynamicStopLoss.maxPercent")))*futuresTypeSign
	}
	if maxHighInPercent < viper.GetFloat64("orders.dynamicStopLoss.minPercent") {
		return currentPrice + int64(util.CalculatePercentOf(float64(currentPrice), viper.GetFloat64("orders.dynamicStopLoss.minPercent")))*futuresTypeSign
	}

	return localExtremum + int64(util.CalculatePercentOf(float64(localExtremum), viper.GetFloat64("orders.dynamicStopLoss.deviationPercent")))*futuresTypeSign
}
