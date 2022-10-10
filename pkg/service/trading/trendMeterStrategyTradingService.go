package trading

import (
	"cryptoBot/pkg/constants"
	"cryptoBot/pkg/constants/bybit"
	"cryptoBot/pkg/constants/futureType"
	constantIndicator "cryptoBot/pkg/constants/indicator"
	"cryptoBot/pkg/data/domains"
	"cryptoBot/pkg/repository"
	"cryptoBot/pkg/service/date"
	"cryptoBot/pkg/service/exchange"
	"cryptoBot/pkg/service/indicator"
	"cryptoBot/pkg/service/orders"
	"cryptoBot/pkg/util"
	"github.com/sdcoffey/big"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"time"
)

var trendMeterStrategyTradingServiceImpl *TrendMeterStrategyTradingService

func NewTrendMeterStrategyTradingService(
	transactionRepo repository.Transaction,
	clock date.Clock,
	exchangeDataService *exchange.DataService,
	klineRepo repository.Kline,
	standardDeviationService *indicator.StandardDeviationService,
	klinesFetcherService *exchange.KlinesFetcherService,
	macdService *indicator.MACDService,
	relativeStrengthIndexService *indicator.RelativeStrengthIndexService,
	exponentialMovingAverageService *indicator.ExponentialMovingAverageService,
	profitLossFinderService *orders.ProfitLossFinderService,
	orderManagerService *orders.OrderManagerService,
	priceChangeTrackingService *orders.PriceChangeTrackingService,
) *TrendMeterStrategyTradingService {
	if trendMeterStrategyTradingServiceImpl != nil {
		panic("Unexpected try to create second service instance")
	}
	trendMeterStrategyTradingServiceImpl = &TrendMeterStrategyTradingService{
		klineRepo:                       klineRepo,
		transactionRepo:                 transactionRepo,
		Clock:                           clock,
		ExchangeDataService:             exchangeDataService,
		StandardDeviationService:        standardDeviationService,
		KlinesFetcherService:            klinesFetcherService,
		MACDService:                     macdService,
		RelativeStrengthIndexService:    relativeStrengthIndexService,
		ExponentialMovingAverageService: exponentialMovingAverageService,
		ProfitLossFinderService:         profitLossFinderService,
		OrderManagerService:             orderManagerService,
		PriceChangeTrackingService:      priceChangeTrackingService,
	}
	return trendMeterStrategyTradingServiceImpl
}

type TrendMeterStrategyTradingService struct {
	transactionRepo                 repository.Transaction
	klineRepo                       repository.Kline
	Clock                           date.Clock
	ExchangeDataService             *exchange.DataService
	StandardDeviationService        *indicator.StandardDeviationService
	KlinesFetcherService            *exchange.KlinesFetcherService
	MACDService                     *indicator.MACDService
	RelativeStrengthIndexService    *indicator.RelativeStrengthIndexService
	ExponentialMovingAverageService *indicator.ExponentialMovingAverageService
	ProfitLossFinderService         *orders.ProfitLossFinderService
	OrderManagerService             *orders.OrderManagerService
	PriceChangeTrackingService      *orders.PriceChangeTrackingService
}

func (s *TrendMeterStrategyTradingService) InitializeTrading(coin *domains.Coin) error {
	err := s.OrderManagerService.SetFuturesLeverage(coin, viper.GetInt("strategy.trendMeter.futures.leverage"))
	if err != nil {
		return err
	}
	return nil
}

func (s *TrendMeterStrategyTradingService) BotAction(coin *domains.Coin) {
	if s.fetchActualKlines(coin) {
		return
	}

	s.BotSingleAction(coin)
}

func (s *TrendMeterStrategyTradingService) fetchActualKlines(coin *domains.Coin) bool {
	lastKline, err := s.klineRepo.FindLast(coin.Id, viper.GetString("strategy.trendMeter.interval"))
	if err != nil {
		zap.S().Errorf("Error FindLast %s", err.Error())
		return true
	}
	var fetchKlinesFrom time.Time
	if lastKline == nil {
		fetchKlinesFrom = s.Clock.NowTime().Add(time.Minute * time.Duration(viper.GetInt("strategy.trendMeter.interval")) * (bybit.BYBIT_MAX_LIMIT) * (-1))
	} else {
		fetchKlinesFrom = lastKline.OpenTime
		if s.Clock.NowTime().Before(lastKline.CloseTime) {
			return true
		}
	}

	if err := s.KlinesFetcherService.FetchKlinesForPeriod(coin, fetchKlinesFrom, s.Clock.NowTime(), viper.GetString("strategy.trendMeter.interval")); err != nil {
		zap.S().Errorf("Error during fetchKlinesForPeriod %s", err.Error())
		return true
	}
	return false
}

func (s *TrendMeterStrategyTradingService) BotSingleAction(coin *domains.Coin) {
	if s.Clock.NowTime().Minute()%viper.GetInt("strategy.trendMeter.interval") == 0 {
		openedOrder, _ := s.transactionRepo.FindOpenedTransaction(constants.TREND_METER)
		//if openedOrder != nil {
		//	s.closeByRealStopLossOrTakeProfit(coin, openedOrder)
		//}

		openedOrder, _ = s.transactionRepo.FindOpenedTransaction(constants.TREND_METER)
		if openedOrder != nil && s.OrderManagerService.ShouldCloseByBreakEven(coin, openedOrder) {
			zap.S().Infof("Close by breakeven at %v \n", s.Clock.NowTime())
			s.OrderManagerService.CloseOrder(openedOrder, coin, openedOrder.Price)
		}

		openedOrder, _ = s.transactionRepo.FindOpenedTransaction(constants.TREND_METER)
		if openedOrder != nil && s.OrderManagerService.ShouldCloseByTrailingTakeProfit(coin, openedOrder) {
			zap.S().Infof("Close by trailing take profit at %v \n", s.Clock.NowTime())
			currentPrice, _ := s.ExchangeDataService.GetCurrentPrice(coin)
			s.OrderManagerService.CloseOrder(openedOrder, coin, currentPrice)
		}

		//openedOrder, _ = s.transactionRepo.FindOpenedTransaction(constants.TREND_METER)
		//if openedOrder != nil {
		//	s.closeOnCrossMA(coin, openedOrder)
		//}

		openedOrder, _ = s.transactionRepo.FindOpenedTransaction(constants.TREND_METER)
		if openedOrder != nil {
			s.checkIfClosedByDynamicStopLoss(coin, openedOrder)
		}

		//openedOrder, _ = s.transactionRepo.FindOpenedTransaction(constants.TREND_METER)
		//if openedOrder != nil {
		//	if s.isTakeProfitSignal(coin, openedOrder) {
		//		currentPrice, _ := s.ExchangeDataService.GetCurrentPrice(coin)
		//		s.OrderManagerService.CloseOrder(openedOrder, coin, currentPrice)
		//	}
		//}

		//openedOrder, _ = s.transactionRepo.FindOpenedTransaction(constants.TREND_METER)
		//if openedOrder != nil {
		//	if s.isStopLossSignal(coin, openedOrder) {
		//		currentPrice, _ := s.ExchangeDataService.GetCurrentPrice(coin)
		//		s.OrderManagerService.CloseOrder(openedOrder, coin, currentPrice)
		//	}
		//}

		openedOrder, _ = s.transactionRepo.FindOpenedTransaction(constants.TREND_METER)
		if openedOrder == nil {
			s.calculateIndicators(coin)
		}
	}
}

func (s *TrendMeterStrategyTradingService) closeOnCrossMA(coin *domains.Coin, openedOrder *domains.Transaction) {
	currentPrice, err := s.ExchangeDataService.GetCurrentPrice(coin)
	if err != nil {
		zap.S().Errorf("Error on ExchangeDataService.GetCurrentPrice: %s", err)
		return
	}

	emaAtOrderOpened := s.ExponentialMovingAverageService.CalculateEmaAtMoment(coin,
		viper.GetString("strategy.trendMeter.interval"), viper.GetInt("strategy.trendMeter.emaSlowLength"),
		openedOrder.CreatedAt)
	emaAtOrderOpenedInCents := int64(emaAtOrderOpened.Float() * 100)

	if openedOrder.FuturesType == futureType.LONG {
		if currentPrice <= emaAtOrderOpenedInCents {
			zap.S().Infof("At %v close LONG  below MA price=%v  emaAtOrderOpened=%v \n", s.Clock.NowTime(), currentPrice, emaAtOrderOpenedInCents)
			s.OrderManagerService.CloseOrder(openedOrder, coin, emaAtOrderOpenedInCents)
		}
	}

	if openedOrder.FuturesType == futureType.SHORT {
		if currentPrice >= emaAtOrderOpenedInCents {
			zap.S().Infof("At %v close SHORT above MA price=%v  emaAtOrderOpened=%v \n", s.Clock.NowTime(), currentPrice, emaAtOrderOpenedInCents)
			s.OrderManagerService.CloseOrder(openedOrder, coin, emaAtOrderOpenedInCents)
		}
	}
}

func (s *TrendMeterStrategyTradingService) checkIfClosedByDynamicStopLoss(coin *domains.Coin, openedOrder *domains.Transaction) {
	//todo: fetch order details and update domain

	allKlinesSinceOrderCreated, err := s.klineRepo.FindAllByCoinIdAndIntervalAndCloseTimeInRange(coin.Id,
		viper.GetString("strategy.trendMeter.interval"), openedOrder.CreatedAt, s.Clock.NowTime())

	if err != nil {
		zap.S().Errorf("Error %s", err.Error())
		return
	}

	stopLossPrice, err := s.ProfitLossFinderService.FindStopLoss(coin, openedOrder.CreatedAt, viper.GetString("strategy.trendMeter.interval"), openedOrder.FuturesType)

	if err != nil {
		zap.S().Errorf("Error %s", err.Error())
		return
	}

	for _, kline := range allKlinesSinceOrderCreated {
		if openedOrder.FuturesType == futureType.LONG && kline.Low < stopLossPrice ||
			openedOrder.FuturesType == futureType.SHORT && kline.High > stopLossPrice {
			s.OrderManagerService.CloseOrder(openedOrder, coin, stopLossPrice)
		}
	}
}

func (s *TrendMeterStrategyTradingService) closeByRealStopLossOrTakeProfit(coin *domains.Coin, openedOrder *domains.Transaction) {
	klines, err := s.klineRepo.FindAllByCoinIdAndIntervalAndCloseTimeInRange(coin.Id, viper.GetString("strategy.trendMeter.interval"), openedOrder.CreatedAt, s.Clock.NowTime())
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

func (s *TrendMeterStrategyTradingService) closeByRealStopLoss(coin *domains.Coin, openedOrder *domains.Transaction, minLow int64, maxHigh int64) bool {
	stopLossPercent := 0.5
	if openedOrder.FuturesType == futureType.LONG {
		stopLossPrice := openedOrder.Price - int64(util.CalculatePercentOf(float64(openedOrder.Price), stopLossPercent))
		if minLow <= stopLossPrice {
			zap.S().Infof(" close by stop loss %v opened=%v stopLossPrice=%v ", futureType.GetString(openedOrder.FuturesType), openedOrder.Price, stopLossPrice)
			s.OrderManagerService.CloseOrder(openedOrder, coin, stopLossPrice)
			return true
		}
	} else {
		stopLossPrice := openedOrder.Price + int64(util.CalculatePercentOf(float64(openedOrder.Price), stopLossPercent))
		if maxHigh >= stopLossPrice {
			zap.S().Infof(" close by stop loss %v opened=%v stopLossPrice=%v ", futureType.GetString(openedOrder.FuturesType), openedOrder.Price, stopLossPrice)
			s.OrderManagerService.CloseOrder(openedOrder, coin, stopLossPrice)
			return true
		}
	}
	return false
}

func (s *TrendMeterStrategyTradingService) closeByRealTakeProfit(coin *domains.Coin, openedOrder *domains.Transaction, minLow int64, maxHigh int64) {
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
		s.OrderManagerService.CloseOrder(openedOrder, coin, takeProfitPrice)
	}
}

func (s *TrendMeterStrategyTradingService) isStopLossSignal(coin *domains.Coin, openedOrder *domains.Transaction) bool {
	currentPrice, err := s.ExchangeDataService.GetCurrentPrice(coin)
	if err != nil {
		zap.S().Errorf("Error during GetCurrentPrice %s", err.Error())
		return false
	}

	profitInPercent := util.CalculateProfitInPercent(openedOrder.Price, currentPrice, openedOrder.FuturesType)
	zap.S().Infof(" profitInPercent %v", profitInPercent)
	if profitInPercent <= -1.0 {
		return true
	}
	return false
}

func (s *TrendMeterStrategyTradingService) isTakeProfitSignal(coin *domains.Coin, openedOrder *domains.Transaction) bool {
	currentPrice, err := s.ExchangeDataService.GetCurrentPrice(coin)
	if err != nil {
		zap.S().Errorf("Error during GetCurrentPrice %s", err.Error())
		return false
	}

	profitInPercent := util.CalculateProfitInPercent(openedOrder.Price, currentPrice, openedOrder.FuturesType)
	if profitInPercent <= viper.GetFloat64("strategy.trendMeter.takeProfit.min") {
		return false
	}

	macdResult := s.MACDService.CalculateCurrentMACD(coin,
		viper.GetString("strategy.trendMeter.interval"),
		viper.GetInt("strategy.trendMeter.trendMeter1.macd.fastLength"),
		viper.GetInt("strategy.trendMeter.trendMeter1.macd.slowLength"),
		viper.GetInt("strategy.trendMeter.trendMeter1.macd.signalLength"))

	return macdResult.Mul(futureType.GetFuturesSignDecimal(openedOrder.FuturesType)).GTE(big.ZERO)
}

func (s *TrendMeterStrategyTradingService) calculateIndicators(coin *domains.Coin) {

	macdSignal, macdFuturesType := s.CalculateMacdSignal(coin)

	rsi13Signal, rs13FuturesType := s.CalculateRsiSignal(coin, viper.GetInt("strategy.trendMeter.trendMeter2.rsi.length"), viper.GetFloat64("strategy.trendMeter.trendMeter2.rsi.signalPoint"))

	if macdFuturesType != rs13FuturesType {
		return
	}

	rsi5Signal, rs5FuturesType := s.CalculateRsiSignal(coin, viper.GetInt("strategy.trendMeter.trendMeter3.rsi.length"), viper.GetFloat64("strategy.trendMeter.trendMeter3.rsi.signalPoint"))

	if rs13FuturesType != rs5FuturesType {
		return
	}

	trendBar1 := s.ExponentialMovingAverageService.IsFastEmaAbove(coin, viper.GetString("strategy.trendMeter.interval"), viper.GetInt("strategy.trendMeter.trendBar1.fastLength"), constantIndicator.EMA, viper.GetInt("strategy.trendMeter.trendBar1.slowLength"), constantIndicator.EMA)

	if futureType.GetFuturesSign(rs5FuturesType) < 0 && trendBar1 {
		return
	}

	trendBar2 := s.ExponentialMovingAverageService.IsFastEmaAbove(coin, viper.GetString("strategy.trendMeter.interval"), viper.GetInt("strategy.trendMeter.trendBar2.fastLength"), constantIndicator.EMA, viper.GetInt("strategy.trendMeter.trendBar2.slowLength"), constantIndicator.SMA)

	if trendBar1 != trendBar2 {
		return
	}

	emaFastAbove := s.ExponentialMovingAverageService.IsFastEmaAbove(coin, viper.GetString("strategy.trendMeter.interval"), viper.GetInt("strategy.trendMeter.emaFastLength"), constantIndicator.EMA, viper.GetInt("strategy.trendMeter.emaSlowLength"), constantIndicator.EMA)

	if trendBar2 != emaFastAbove {
		return
	}

	volatilityOscillatorSignal, volatilityFuturesType := s.StandardDeviationService.IsVolatilityOscillatorSignal(coin, viper.GetString("strategy.trendMeter.interval"))

	trendMeterSignalLong := (macdSignal || rsi13Signal || rsi5Signal) && macdFuturesType == futureType.LONG && rs13FuturesType == futureType.LONG && rs5FuturesType == futureType.LONG
	trendMeterSignalShort := (macdSignal || rsi13Signal || rsi5Signal) && macdFuturesType == futureType.SHORT && rs13FuturesType == futureType.SHORT && rs5FuturesType == futureType.SHORT

	//if > 10% start
	currentPrice, _ := s.ExchangeDataService.GetCurrentPrice(coin)
	currentEMA := s.ExponentialMovingAverageService.CalculateCurrentEMA(coin, viper.GetString("strategy.trendMeter.interval"), viper.GetInt("strategy.trendMeter.emaSlowLength"))
	currentEmaInt := int64(currentEMA.Float() * 100)
	changedFromEmaInPercent := util.CalculateChangeInPercents(currentPrice, currentEmaInt)
	if changedFromEmaInPercent > 10 {
		zap.S().Infof("DO NOT OPEN ORDER changedFromEmaInPercent=%v", currentEmaInt)
		return
	}
	//if > 10% end

	if trendMeterSignalLong && trendBar1 && trendBar2 && emaFastAbove && volatilityOscillatorSignal && volatilityFuturesType == futureType.LONG {
		zap.S().Infof("OPEN LONG SIGNAL")
		s.openOrder(coin, futureType.LONG)
	}
	if trendMeterSignalShort && !trendBar1 && !trendBar2 && !emaFastAbove && volatilityOscillatorSignal && volatilityFuturesType == futureType.SHORT {
		zap.S().Infof("OPEN SHORT SIGNAL")
		s.openOrder(coin, futureType.SHORT)
	}

	zap.S().Debugf("Signal trendMeter1MACD %v for %v", macdSignal, futureType.GetString(macdFuturesType))
	zap.S().Debugf("Signal trendMeter2RSI13 %v for %v", rsi13Signal, futureType.GetString(rs13FuturesType))
	zap.S().Debugf("Signal trendMeter3RSI5  %v for %v", rsi5Signal, futureType.GetString(rs5FuturesType))
	zap.S().Debugf("trendBar1 %v", futureType.GetString(futureType.GetTypeByBool(trendBar1)))
	zap.S().Debugf("trendBar2 %v", futureType.GetString(futureType.GetTypeByBool(trendBar2)))
	zap.S().Debugf("emaIndicator %v", futureType.GetString(futureType.GetTypeByBool(emaFastAbove)))
	zap.S().Debugf("volatilityOscillatorSignal=%v for %v", volatilityOscillatorSignal, futureType.GetString(volatilityFuturesType))
	zap.S().Debugf("")

}

func (s *TrendMeterStrategyTradingService) openOrder(coin *domains.Coin, futuresType futureType.FuturesType) {
	stopLossPrice, err := s.ProfitLossFinderService.FindStopLoss(coin, s.Clock.NowTime(), viper.GetString("strategy.trendMeter.interval"), futuresType)

	if err != nil {
		zap.S().Errorf("Error %s", err.Error())
		return
	}

	s.OrderManagerService.OpenOrderWithFixedStopLoss(coin, futuresType, stopLossPrice)
}

// CalculateMacdSignal signal is true when MACD cross the ZERO value (was < 0, now > 0 and the opposite)
func (s *TrendMeterStrategyTradingService) CalculateMacdSignal(coin *domains.Coin) (bool, futureType.FuturesType) {
	macdList := s.MACDService.CalculateMACDForAll(coin,
		viper.GetString("strategy.trendMeter.interval"),
		viper.GetInt("strategy.trendMeter.trendMeter1.macd.fastLength"),
		viper.GetInt("strategy.trendMeter.trendMeter1.macd.slowLength"),
		viper.GetInt("strategy.trendMeter.trendMeter1.macd.signalLength"))

	prevMacdValue := macdList[(len(macdList) - 2)]
	currMacdValue := macdList[(len(macdList) - 1)]

	return prevMacdValue.Mul(currMacdValue).LT(big.ZERO), futureType.GetTypeByBool(currMacdValue.GT(big.ZERO))
}

// CalculateRsiSignal is true when RSI changes from Long to Short and the opposite
func (s *TrendMeterStrategyTradingService) CalculateRsiSignal(coin *domains.Coin, rsiLength int, rsiSignalPoint float64) (bool, futureType.FuturesType) {
	trendMeterRSI := s.RelativeStrengthIndexService.CalculateRSIForAll(coin,
		viper.GetString("strategy.trendMeter.interval"), rsiLength)

	prevIsLong := trendMeterRSI[(len(trendMeterRSI) - 2)].GTE(big.NewDecimal(rsiSignalPoint))
	currIsLong := trendMeterRSI[(len(trendMeterRSI) - 1)].GTE(big.NewDecimal(rsiSignalPoint))

	return prevIsLong != currIsLong, futureType.GetTypeByBool(currIsLong)
}
