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
	"fmt"
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
	orderManagerService *orders.OrderManagerService,
	priceChangeTrackingService *orders.PriceChangeTrackingService,
) *TrendMeterStrategyTradingService {
	if trendMeterStrategyTradingServiceImpl != nil {
		panic("Unexpected try to create second service instance")
	}
	trendMeterStrategyTradingServiceImpl = &TrendMeterStrategyTradingService{
		KlineRepo:                       klineRepo,
		TransactionRepo:                 transactionRepo,
		Clock:                           clock,
		ExchangeDataService:             exchangeDataService,
		StandardDeviationService:        standardDeviationService,
		KlinesFetcherService:            klinesFetcherService,
		MACDService:                     macdService,
		RelativeStrengthIndexService:    relativeStrengthIndexService,
		ExponentialMovingAverageService: exponentialMovingAverageService,
		OrderManagerService:             orderManagerService,
		PriceChangeTrackingService:      priceChangeTrackingService,
	}
	return trendMeterStrategyTradingServiceImpl
}

type TrendMeterStrategyTradingService struct {
	TransactionRepo                 repository.Transaction
	KlineRepo                       repository.Kline
	Clock                           date.Clock
	ExchangeDataService             *exchange.DataService
	StandardDeviationService        *indicator.StandardDeviationService
	KlinesFetcherService            *exchange.KlinesFetcherService
	MACDService                     *indicator.MACDService
	RelativeStrengthIndexService    *indicator.RelativeStrengthIndexService
	ExponentialMovingAverageService *indicator.ExponentialMovingAverageService
	OrderManagerService             *orders.OrderManagerService
	PriceChangeTrackingService      *orders.PriceChangeTrackingService
}

func (s *TrendMeterStrategyTradingService) InitializeTrading(coin *domains.Coin) error {
	err := s.OrderManagerService.SetFuturesLeverage(coin, viper.GetInt("strategy.trendMeter.futures.leverage"))
	if err != nil {
		return err
	}

	s.fetchActualKlines(coin, viper.GetInt("strategy.trendMeter.interval"))
	s.fetchActualKlines(coin, 1)

	return nil
}

func (s *TrendMeterStrategyTradingService) BotAction(coin *domains.Coin) {
	s.fetchActualKlines(coin, viper.GetInt("strategy.trendMeter.interval"))
	s.fetchActualKlines(coin, 1)

	s.BotActionCheckIfOrderClosedByExchange(coin)

	s.BotActionCloseOrderIfNeeded(coin)

	if s.Clock.NowTime().Minute()%viper.GetInt("strategy.trendMeter.interval") != 0 {
		return
	}

	s.BotActionOpenOrderIfNeeded(coin)
}

func (s *TrendMeterStrategyTradingService) fetchActualKlines(coin *domains.Coin, intervalInMinutes int) {
	lastKline, err := s.KlineRepo.FindLast(coin.Id, fmt.Sprint(intervalInMinutes))
	if err != nil {
		zap.S().Errorf("Error FindLast %s", err.Error())
		return
	}
	var fetchKlinesFrom time.Time
	if lastKline == nil {
		fetchKlinesFrom = s.Clock.NowTime().Add(time.Minute * time.Duration(intervalInMinutes) * (bybit.BYBIT_MAX_LIMIT) * (-1))
	} else {
		fetchKlinesFrom = lastKline.OpenTime
		if s.Clock.NowTime().Before(lastKline.CloseTime) {
			return
		}
	}

	if err := s.KlinesFetcherService.FetchKlinesForPeriod(coin, fetchKlinesFrom, s.Clock.NowTime(), fmt.Sprint(intervalInMinutes)); err != nil {
		zap.S().Errorf("Error during fetchKlinesForPeriod %s", err.Error())
		return
	}
	return
}

func (s *TrendMeterStrategyTradingService) BotActionCheckIfOrderClosedByExchange(coin *domains.Coin) {
	openedOrder, _ := s.TransactionRepo.FindOpenedTransaction(constants.TREND_METER)
	if openedOrder == nil {
		return
	}

	if isPositionOpened := s.ExchangeDataService.IsPositionOpened(coin, openedOrder); !isPositionOpened && openedOrder != nil {
		s.OrderManagerService.CreateCloseTransactionOnOrderClosedByExchange(coin, openedOrder)
	}

}

func (s *TrendMeterStrategyTradingService) BotActionCloseOrderIfNeeded(coin *domains.Coin) {
	openedOrder, _ := s.TransactionRepo.FindOpenedTransaction(constants.TREND_METER)
	if openedOrder != nil && s.OrderManagerService.ShouldCloseByBreakEven(coin, openedOrder) {
		currentPrice, _ := s.ExchangeDataService.GetCurrentPrice(coin)
		zap.S().Infof("Close by breakeven at %v with price %v \n", s.Clock.NowTime(), currentPrice)
		s.OrderManagerService.CloseOrder(openedOrder, coin, currentPrice)
	}

	openedOrder, _ = s.TransactionRepo.FindOpenedTransaction(constants.TREND_METER)
	if openedOrder != nil {
		if s.isTakeProfitSignal(coin, openedOrder) {
			currentPrice, _ := s.ExchangeDataService.GetCurrentPrice(coin)
			s.OrderManagerService.CloseOrder(openedOrder, coin, currentPrice)
		}
	}
}

func (s *TrendMeterStrategyTradingService) BotActionOpenOrderIfNeeded(coin *domains.Coin) {
	openedOrder, _ := s.TransactionRepo.FindOpenedTransaction(constants.TREND_METER)

	if openedOrder != nil {
		return
	}

	s.calculateIndicators(coin)
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
	changedFromEmaInPercent := util.CalculateChangeInPercentsAbs(currentEmaInt, currentPrice)
	if changedFromEmaInPercent > 10 {
		zap.S().Infof("DO NOT OPEN ORDER - FAR FROM EMA - changedFromEmaInPercent=%v", currentEmaInt)
		return
	}
	if changedFromEmaInPercent < 2.5 {
		zap.S().Infof("DO NOT OPEN ORDER - CLOSE TO EMA - changedFromEmaInPercent=%v", currentEmaInt)
		return
	}
	//if > 10% end

	if trendMeterSignalLong && trendBar1 && trendBar2 && emaFastAbove && volatilityOscillatorSignal && volatilityFuturesType == futureType.LONG {
		s.OrderManagerService.OpenOrderWithCalculateStopLoss(coin, futureType.LONG, viper.GetString("strategy.trendMeter.interval"))
	}
	if trendMeterSignalShort && !trendBar1 && !trendBar2 && !emaFastAbove && volatilityOscillatorSignal && volatilityFuturesType == futureType.SHORT {
		s.OrderManagerService.OpenOrderWithCalculateStopLoss(coin, futureType.SHORT, viper.GetString("strategy.trendMeter.interval"))
	}
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
