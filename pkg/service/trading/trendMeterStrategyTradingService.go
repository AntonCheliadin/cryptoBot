package trading

import (
	telegramApi "cryptoBot/pkg/api/telegram"
	"cryptoBot/pkg/constants"
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
	tradingType constants.TradingType,
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
		tradingType:                     tradingType,
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
	tradingType                     constants.TradingType
}

func (s *TrendMeterStrategyTradingService) InitializeTrading(coin *domains.Coin) error {
	if s.tradingType == constants.FUTURES {
		err := s.OrderManagerService.SetFuturesLeverage(coin, viper.GetInt("strategy.trendMeter.futures.leverage"))
		if err != nil {
			return err
		}
	}

	s.KlinesFetcherService.FetchActualKlines(coin, viper.GetInt("strategy.trendMeter.interval"))
	s.KlinesFetcherService.FetchActualKlines(coin, 1)

	return nil
}

func (s *TrendMeterStrategyTradingService) BotAction(coin *domains.Coin) {
	s.KlinesFetcherService.FetchActualKlines(coin, viper.GetInt("strategy.trendMeter.interval"))
	s.KlinesFetcherService.FetchActualKlines(coin, 1)

	s.BotActionCheckIfOrderClosedByExchange(coin)

	if s.Clock.NowTime().Minute()%viper.GetInt("strategy.trendMeter.interval") != 0 {
		return
	}

	s.BotActionBuyMoreIfNeeded(coin)
	s.BotActionCloseOrderIfNeeded(coin)

	s.BotActionOpenOrderIfNeeded(coin)
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

func (s *TrendMeterStrategyTradingService) BotActionBuyMoreIfNeeded(coin *domains.Coin) {
	openedOrders, _ := s.TransactionRepo.FindAllOpenedTransactions(constants.TREND_METER)
	openedTransactionsCount := len(openedOrders)
	if openedTransactionsCount == 0 {
		return
	}

	currentPrice, err := s.ExchangeDataService.GetCurrentPrice(coin)
	if err != nil {
		zap.S().Errorf("Error during GetCurrentPrice %s", err.Error())
		return
	}

	firstTransaction := openedOrders[0]
	profitInPercent := util.CalculateProfitInPercent(firstTransaction.Price, currentPrice, futureType.LONG)

	if profitInPercent > -10 {
		return
	}

	costInUSDT := int64(0)

	if openedTransactionsCount == 9 && profitInPercent < -90 {
		costInUSDT = int64(3000)
	} else if openedTransactionsCount == 8 && profitInPercent < -80 {
		costInUSDT = int64(2800)
	} else if openedTransactionsCount == 7 && profitInPercent < -70 {
		costInUSDT = int64(2400)
	} else if openedTransactionsCount == 6 && profitInPercent < -60 {
		costInUSDT = int64(2000)
	} else if openedTransactionsCount == 5 && profitInPercent < -50 {
		costInUSDT = int64(1000)
	} else if openedTransactionsCount == 4 && profitInPercent < -40 {
		costInUSDT = int64(500)
	} else if openedTransactionsCount == 3 && profitInPercent < -30 {
		costInUSDT = int64(1000)
	} else if openedTransactionsCount == 2 && profitInPercent < -20 {
		costInUSDT = int64(300)
	} else if openedTransactionsCount == 1 && profitInPercent < -10 {
		costInUSDT = int64(200)
	} else {
		return
	}

	s.OrderManagerService.OpenOrderWithCost(coin, "", futureType.LONG, float64(costInUSDT), s.tradingType)
}

func (s *TrendMeterStrategyTradingService) BotActionCloseOrderIfNeeded(coin *domains.Coin) {
	openedOrders, _ := s.TransactionRepo.FindAllOpenedTransactions(constants.TREND_METER)
	if len(openedOrders) == 1 {
		openedOrder := openedOrders[0]
		if s.isTakeProfitSignal(coin, openedOrder) {
			currentPrice, _ := s.ExchangeDataService.GetCurrentPrice(coin)
			s.OrderManagerService.CloseOrder(openedOrder, coin, currentPrice, s.tradingType)
		}
	} else if len(openedOrders) > 1 {
		if s.isTakeProfitSignalForCombinedOrder(coin, openedOrders) {
			currentPrice, _ := s.ExchangeDataService.GetCurrentPrice(coin)
			s.OrderManagerService.CloseCombinedOrder(openedOrders, coin, currentPrice, s.tradingType)
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

func (s *TrendMeterStrategyTradingService) isTakeProfitSignalForCombinedOrder(coin *domains.Coin, openedTransactions []*domains.Transaction) bool {
	currentPrice, err := s.ExchangeDataService.GetCurrentPrice(coin)
	if err != nil {
		zap.S().Errorf("Error during GetCurrentPrice %s", err.Error())
		return false
	}

	avgPrice := s.calculateAveragePrice(openedTransactions)

	profitInPercent := util.CalculateProfitInPercent(avgPrice, currentPrice, futureType.LONG)

	openedOrders, _ := s.TransactionRepo.FindAllOpenedTransactions(constants.TREND_METER)

	isProfitSignal := profitInPercent > float64(len(openedOrders)-1)
	if isProfitSignal {
		telegramApi.SendTextToTelegramChat(fmt.Sprintf("Close combined order with profit in percent: %s ", profitInPercent))
	}
	return isProfitSignal
}

func (s *TrendMeterStrategyTradingService) calculateAveragePrice(openedTransactions []*domains.Transaction) float64 {
	totalCost := float64(0)
	for _, transaction := range openedTransactions {
		totalCost += transaction.TotalCost
	}

	totalAmount := float64(0)
	for _, transaction := range openedTransactions {
		totalAmount += transaction.Amount
	}

	return (float64(totalCost) / totalAmount)
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

	signDecimal := futureType.GetFuturesSignDecimal(openedOrder.FuturesType)
	return macdResult.Mul(signDecimal).LT(big.ZERO)
}

func (s *TrendMeterStrategyTradingService) calculateIndicators(coin *domains.Coin) {

	macdSignal, macdFuturesType := s.CalculateMacdSignal(coin)

	if macdFuturesType == futureType.SHORT {
		return
	}

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

	////if > 10% start
	//currentPrice, _ := s.ExchangeDataService.GetCurrentPrice(coin)
	//currentEMA := s.ExponentialMovingAverageService.CalculateCurrentEMA(coin, viper.GetString("strategy.trendMeter.interval"), viper.GetInt("strategy.trendMeter.emaSlowLength"))
	//currentEmaInCents := int64(currentEMA.Float() * 100)
	//changedFromEmaInPercent := util.CalculateChangeInPercentsAbs(currentEmaInCents, currentPrice)
	//if changedFromEmaInPercent > 12 {
	//	zap.S().Infof("DO NOT OPEN ORDER - FAR FROM EMA - changedFromEmaInPercent=%v", currentEmaInCents)
	//	return
	//}
	////if > 10% end

	if trendMeterSignalLong && trendBar1 && trendBar2 && emaFastAbove && volatilityOscillatorSignal && volatilityFuturesType == futureType.LONG {
		s.OrderManagerService.OpenOrderWithCost(coin, "", futureType.LONG, util.GetDollarsByCents(viper.GetInt64("strategy.trendMeter.initialCostInCents")), s.tradingType)
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
