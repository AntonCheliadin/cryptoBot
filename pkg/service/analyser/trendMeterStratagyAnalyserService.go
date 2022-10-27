package analyser

import (
	"cryptoBot/pkg/constants"
	"cryptoBot/pkg/constants/futureType"
	"cryptoBot/pkg/data/domains"
	"cryptoBot/pkg/service/date"
	"cryptoBot/pkg/service/trading"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"time"
)

var trendMeterStratagyAnalyserServiceImpl *TrendMeterStratagyAnalyserService

func NewTrendMeterStratagyAnalyserService(tradingService *trading.TrendMeterStrategyTradingService) *TrendMeterStratagyAnalyserService {
	if trendMeterStratagyAnalyserServiceImpl != nil {
		panic("Unexpected try to create second service instance")
	}
	trendMeterStratagyAnalyserServiceImpl = &TrendMeterStratagyAnalyserService{
		tradingService: tradingService,
	}
	return trendMeterStratagyAnalyserServiceImpl
}

type TrendMeterStratagyAnalyserService struct {
	tradingService *trading.TrendMeterStrategyTradingService
}

func (s *TrendMeterStratagyAnalyserService) AnalyseCoin(coin *domains.Coin, from string, to string) {
	timeMax, _ := time.Parse(constants.DATE_FORMAT, to)
	timeIterator, _ := time.Parse(constants.DATE_FORMAT, from)
	timeIterator = timeIterator.Add(time.Second * 2).Add(time.Hour)

	for ; timeIterator.Before(timeMax); timeIterator = timeIterator.Add(time.Minute * 15) {
		clockMock := date.GetClockMock(timeIterator)

		s.tradingService.Clock = clockMock
		s.tradingService.StandardDeviationService.TechanConvertorService.Clock = clockMock
		s.tradingService.ExchangeDataService.Clock = clockMock
		s.tradingService.StandardDeviationService.Clock = clockMock
		s.tradingService.OrderManagerService.ProfitLossFinderService.Clock = clockMock
		s.tradingService.OrderManagerService.Clock = clockMock

		openedOrder, _ := s.tradingService.TransactionRepo.FindOpenedTransaction(constants.TREND_METER)
		if openedOrder != nil {
			s.checkIfClosedByDynamicStopLoss(coin, openedOrder)
		}

		s.tradingService.BotActionCloseOrderIfNeeded(coin)
		s.tradingService.BotActionOpenOrderIfNeeded(coin)
	}
}

func (s *TrendMeterStratagyAnalyserService) checkIfClosedByDynamicStopLoss(coin *domains.Coin, openedOrder *domains.Transaction) {
	allKlinesSinceOrderCreated, err := s.tradingService.KlineRepo.FindAllByCoinIdAndIntervalAndCloseTimeInRange(coin.Id,
		viper.GetString("strategy.trendMeter.interval"), openedOrder.CreatedAt, s.tradingService.Clock.NowTime())

	if err != nil {
		zap.S().Errorf("Error %s", err.Error())
		return
	}

	stopLossPrice, err := s.tradingService.OrderManagerService.ProfitLossFinderService.FindStopLoss(coin, openedOrder.CreatedAt, viper.GetString("strategy.trendMeter.interval"), openedOrder.FuturesType)

	if err != nil {
		zap.S().Errorf("Error %s", err.Error())
		return
	}

	for _, kline := range allKlinesSinceOrderCreated {
		if openedOrder.FuturesType == futureType.LONG && kline.Low < stopLossPrice ||
			openedOrder.FuturesType == futureType.SHORT && kline.High > stopLossPrice {
			s.tradingService.OrderManagerService.CloseOrder(openedOrder, coin, stopLossPrice)
		}
	}
}

/*  closeByRealStopLossOrTakeProfit
func (s *TrendMeterStratagyAnalyserService) closeByRealStopLossOrTakeProfit(coin *domains.Coin, openedOrder *domains.Transaction) {
	klines, err := s.tradingService.klineRepo.FindAllByCoinIdAndIntervalAndCloseTimeInRange(coin.Id, viper.GetString("strategy.trendMeter.interval"), openedOrder.CreatedAt, s.Clock.NowTime())
	if err != nil {
		return
	}

	maxHigh := int64(0)
	minLow := int64(9223372036854775807)

	for _, kline := range klines {
		maxHigh = util.Max(maxHigh, kline.High)
		minLow = util.Min(minLow, kline.Low)
	}

	if s.closeByRealStopLoss(coin, openedOrder, minLow, maxHigh) {
		return
	}

	s.closeByRealTakeProfit(coin, openedOrder, minLow, maxHigh)
}

func (s *TrendMeterStratagyAnalyserService) closeByRealStopLoss(coin *domains.Coin, openedOrder *domains.Transaction, minLow int64, maxHigh int64) bool {
	stopLossPercent := 0.5
	if openedOrder.FuturesType == futureType.LONG {
		stopLossPrice := openedOrder.Price - int64(util.CalculatePercentOf(float64(openedOrder.Price), stopLossPercent))
		if minLow <= stopLossPrice {
			zap.S().Infof(" close by stop loss %v opened=%v stopLossPrice=%v ", futureType.GetString(openedOrder.FuturesType), openedOrder.Price, stopLossPrice)
			s.tradingService.OrderManagerService.CloseOrder(openedOrder, coin, stopLossPrice)
			return true
		}
	} else {
		stopLossPrice := openedOrder.Price + int64(util.CalculatePercentOf(float64(openedOrder.Price), stopLossPercent))
		if maxHigh >= stopLossPrice {
			zap.S().Infof(" close by stop loss %v opened=%v stopLossPrice=%v ", futureType.GetString(openedOrder.FuturesType), openedOrder.Price, stopLossPrice)
			s.tradingService.OrderManagerService.CloseOrder(openedOrder, coin, stopLossPrice)
			return true
		}
	}
	return false
}

func (s *TrendMeterStratagyAnalyserService) closeByRealTakeProfit(coin *domains.Coin, openedOrder *domains.Transaction, minLow int64, maxHigh int64) {
	takeProfitExtremum := minLow
	if openedOrder.FuturesType == futureType.LONG {
		takeProfitExtremum = maxHigh
	}

	takeProfitPercent := 1.0
	profitInPercent := util.CalculateProfitInPercent(openedOrder.Price, takeProfitExtremum, openedOrder.FuturesType)

	zap.S().Infof(" profitInPercent %v %v opened=%v takeProfitExtremum=%v ",
		profitInPercent, futureType.GetString(openedOrder.FuturesType), openedOrder.Price, takeProfitExtremum)

	if profitInPercent >= takeProfitPercent {
		takeProfitPrice := openedOrder.Price + int64(util.CalculatePercentOf(float64(openedOrder.Price), takeProfitPercent)*futureType.GetFuturesSignFloat64(openedOrder.FuturesType))
		zap.S().Infof(" close by take profit %v profitInPercent=%v takeProfitPrice=%v", futureType.GetString(openedOrder.FuturesType), profitInPercent, takeProfitPrice)
		s.tradingService.OrderManagerService.CloseOrder(openedOrder, coin, takeProfitPrice)
	}
}*/
