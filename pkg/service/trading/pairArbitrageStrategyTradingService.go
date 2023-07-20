package trading

import (
	telegramApi "cryptoBot/pkg/api/telegram"
	"cryptoBot/pkg/constants"
	"cryptoBot/pkg/constants/futureType"
	"cryptoBot/pkg/data/domains"
	"cryptoBot/pkg/repository"
	"cryptoBot/pkg/service/date"
	"cryptoBot/pkg/service/exchange"
	"cryptoBot/pkg/service/indicator/techanLib"
	"cryptoBot/pkg/service/orders"
	"cryptoBot/pkg/util"
	"fmt"
	"github.com/sdcoffey/big"
	"github.com/sdcoffey/techan"
	"go.uber.org/zap"
)

//https://youtu.be/9jn3DnLNyU0
//Z-Score script: https://www.tradingview.com/pine/?id=PUB%3BC0yY0a1BOlCTSIHGTDWwBkWcwTdjpeEd
var pairArbitrageStrategyTradingService *PairArbitrageStrategyTradingService

func NewPairArbitrageStrategyTradingService(
	transactionRepo repository.Transaction,
	clock date.Clock,
	exchangeDataService *exchange.DataService,
	syntheticKlineRepo repository.SyntheticKline,
	klinesFetcherService *exchange.KlinesFetcherService,
	orderManagerService *orders.OrderManagerService,
	techanConvertorService *techanLib.TechanConvertorService,
	coin1 *domains.Coin,
	coin2 *domains.Coin,
) *PairArbitrageStrategyTradingService {
	pairArbitrageStrategyTradingService = &PairArbitrageStrategyTradingService{
		SyntheticKlineRepo:     syntheticKlineRepo,
		TransactionRepo:        transactionRepo,
		Clock:                  clock,
		ExchangeDataService:    exchangeDataService,
		KlinesFetcherService:   klinesFetcherService,
		OrderManagerService:    orderManagerService,
		TechanConvertorService: techanConvertorService,
		coin1:                  coin1,
		coin2:                  coin2,
		startCapitalInCents:    10000,
		strategyLength:         20,
		klineInterval:          60,
		klineIntervalS:         "60",
		leverage:               1,
		stopLossPercent:        0, //disabled
		closeOnProfit:          1,
		maxOrderLoss:           -3,
		tradingStrategy:        constants.PAIR_ARBITRAGE,
	}
	return pairArbitrageStrategyTradingService
}

type PairArbitrageStrategyTradingService struct {
	TransactionRepo        repository.Transaction
	SyntheticKlineRepo     repository.SyntheticKline
	Clock                  date.Clock
	ExchangeDataService    *exchange.DataService
	KlinesFetcherService   *exchange.KlinesFetcherService
	OrderManagerService    *orders.OrderManagerService
	TechanConvertorService *techanLib.TechanConvertorService
	coin1                  *domains.Coin
	coin2                  *domains.Coin
	startCapitalInCents    int
	strategyLength         int
	klineInterval          int
	klineIntervalS         string
	leverage               int
	stopLossPercent        float64
	closeOnProfit          float64
	maxOrderLoss           float64
	tradingStrategy        constants.TradingStrategy
}

func (s *PairArbitrageStrategyTradingService) BotAction(coin *domains.Coin) {
	return
}
func (s *PairArbitrageStrategyTradingService) InitializeTrading(coin *domains.Coin) error {
	return nil
}

func (s *PairArbitrageStrategyTradingService) Initialize() error {
	err := s.OrderManagerService.SetIsolatedMargin(s.coin1, s.leverage)
	if err != nil {
		return err
	}

	err = s.OrderManagerService.SetIsolatedMargin(s.coin2, s.leverage)
	if err != nil {
		return err
	}

	return nil
}

func (s *PairArbitrageStrategyTradingService) BeforeExecute() {
	if s.Clock.NowTime().Minute()%s.klineInterval != 0 {
		return
	}

	s.KlinesFetcherService.FetchActualKlines(s.coin1, s.klineInterval)
	s.KlinesFetcherService.FetchActualKlines(s.coin2, s.klineInterval)
	s.SyntheticKlineRepo.RefreshView()
}

func (s *PairArbitrageStrategyTradingService) Execute() {
	if s.Clock.NowTime().Minute()%s.klineInterval != 0 {
		return
	}

	klinesFetchLimit := s.strategyLength + 1
	klines, err := s.SyntheticKlineRepo.FindAllByCoinIdAndIntervalAndCloseTimeLessOrderByOpenTimeWithLimit(s.coin1.Id, s.coin2.Id, s.klineIntervalS, s.Clock.NowTime(), klinesFetchLimit)
	if err != nil {
		zap.S().Errorf("Error on fetch synthetic klines: %s. ", err.Error())
		return
	}
	if len(klines) < klinesFetchLimit {
		zap.S().Errorf("Empty klines: %s. ", s.Clock.NowTime().Format(constants.DATE_TIME_FORMAT))
		return
	}

	zScore := s.calculateZScore(klines)

	if s.hasOpenedOrders() {
		s.CloseOpenedOrderByStopLossIfNeeded()
		if zScore.GT(big.NewDecimal(-0.1)) && zScore.LT(big.NewDecimal(0.1)) {
			zap.S().Infof("Close by zScore(%v) crossed at %v", zScore, s.Clock.NowTime().Format(constants.DATE_TIME_FORMAT))
			closedOrder1, closedOrder2 := s.closeOrders()
			telegramApi.SendTextToTelegramChat(fmt.Sprintf("Closed by zScore %v - %v profit: %+d (%.2f%%)", s.coin1.Symbol, s.coin2.Symbol, closedOrder1.Profit.Int64+closedOrder2.Profit.Int64, closedOrder1.PercentProfit.Float64+closedOrder2.PercentProfit.Float64))
		}
		return
	}

	if zScore.GT(big.NewDecimal(2)) {
		zap.S().Infof("Upper Level zScore(%v) crossed at %v", zScore, s.Clock.NowTime().Format(constants.DATE_TIME_FORMAT))
		s.openOrder(s.coin1, futureType.SHORT)
		s.openOrder(s.coin2, futureType.LONG)
		telegramApi.SendTextToTelegramChat("Opened " + s.coin1.Symbol + "⬇️" + s.coin2.Symbol + "⬆ ️")
	} else if zScore.LT(big.NewDecimal(-2)) {
		zap.S().Infof("Lower Level zScore(%v) crossed at %v", zScore, s.Clock.NowTime().Format(constants.DATE_TIME_FORMAT))
		s.openOrder(s.coin1, futureType.LONG)
		s.openOrder(s.coin2, futureType.SHORT)
		telegramApi.SendTextToTelegramChat("Opened " + s.coin1.Symbol + "⬆ ️" + s.coin2.Symbol + "⬇️")
	}
}

func (s *PairArbitrageStrategyTradingService) calculateZScore(klines []domains.IKline) big.Decimal {
	series := s.TechanConvertorService.ConvertKlinesToSeries(klines, s.klineInterval)
	smaIndicator := techan.NewSimpleMovingAverage(techan.NewClosePriceIndicator(series), s.strategyLength)
	stdevIndicator := techan.NewStandardDeviationIndicator(techan.NewClosePriceIndicator(series))

	stdevValue := stdevIndicator.Calculate(s.strategyLength)
	smaValue := smaIndicator.Calculate(s.strategyLength)

	//zsc = (src - sma(src, length)) / selectedStdev
	zScore := (series.LastCandle().ClosePrice.Sub(smaValue)).Div(stdevValue)
	return zScore
}

// CloseOpenedOrderByStopLossIfNeeded if one order closed by stopLoss then close other with current price
func (s *PairArbitrageStrategyTradingService) CloseOpenedOrderByStopLossIfNeeded() {
	openedOrder1, _ := s.TransactionRepo.FindOpenedTransactionByCoin(s.tradingStrategy, s.coin1.Id)
	openedOrder2, _ := s.TransactionRepo.FindOpenedTransactionByCoin(s.tradingStrategy, s.coin2.Id)

	if openedOrder1 != nil {
		isClosed := s.OrderManagerService.CloseOrderByFixedStopLossOrTakeProfit(s.coin1, openedOrder1, s.klineIntervalS)
		if isClosed {
			openedOrder1 = nil
		}
	}
	if openedOrder2 != nil {
		isClosed := s.OrderManagerService.CloseOrderByFixedStopLossOrTakeProfit(s.coin2, openedOrder2, s.klineIntervalS)
		if isClosed {
			openedOrder2 = nil
		}
	}

	//if one of order has been closed by exchange
	if openedOrder1 == nil && openedOrder2 != nil || openedOrder2 == nil && openedOrder1 != nil {
		zap.S().Infof("Order closed by exchange")
		closedOrder1, closedOrder2 := s.closeOrders()
		telegramApi.SendTextToTelegramChat(fmt.Sprintf("Closed by exchange %v - %v profit: %+d (%.2f%%)", s.coin1.Symbol, s.coin2.Symbol, closedOrder1.Profit.Int64+closedOrder2.Profit.Int64, closedOrder1.PercentProfit.Float64+closedOrder2.PercentProfit.Float64))
		return
	}

	currentPrice1, _ := s.ExchangeDataService.GetCurrentPriceForFutures(s.coin1, s.klineInterval)
	currentPrice2, _ := s.ExchangeDataService.GetCurrentPriceForFutures(s.coin2, s.klineInterval)

	profitInPercent1 := util.CalculateProfitInPercent(openedOrder1.Price, currentPrice1, openedOrder1.FuturesType)
	profitInPercent2 := util.CalculateProfitInPercent(openedOrder2.Price, currentPrice2, openedOrder2.FuturesType)

	sumProfit := profitInPercent1 + profitInPercent2
	if sumProfit < s.maxOrderLoss {
		zap.S().Infof("Close orders by stopLoss[%.2f] at %v", sumProfit, s.Clock.NowTime().Format(constants.DATE_TIME_FORMAT))
		closedOrder1, closedOrder2 := s.closeOrders()
		telegramApi.SendTextToTelegramChat(fmt.Sprintf("Closed by stopLoss %v - %v profit: %+d (%.2f%%)", s.coin1.Symbol, s.coin2.Symbol, closedOrder1.Profit.Int64+closedOrder2.Profit.Int64, closedOrder1.PercentProfit.Float64+closedOrder2.PercentProfit.Float64))
		return
	}
	if sumProfit > s.closeOnProfit {
		zap.S().Infof("Close orders with profit[%.2f] at %v", sumProfit, s.Clock.NowTime().Format(constants.DATE_TIME_FORMAT))
		closedOrder1, closedOrder2 := s.closeOrders()
		telegramApi.SendTextToTelegramChat(fmt.Sprintf("Closed with profit %v - %v profit: %+d (%.2f%%)", s.coin1.Symbol, s.coin2.Symbol, closedOrder1.Profit.Int64+closedOrder2.Profit.Int64, closedOrder1.PercentProfit.Float64+closedOrder2.PercentProfit.Float64))
		return
	}
}

func (s *PairArbitrageStrategyTradingService) closeOrders() (*domains.Transaction, *domains.Transaction) {
	zap.S().Infof("Close orders")
	openedOrder1, _ := s.TransactionRepo.FindOpenedTransactionByCoin(s.tradingStrategy, s.coin1.Id)
	var closedOrder1 *domains.Transaction
	if openedOrder1 != nil {
		closedOrder1 = s.OrderManagerService.CloseFuturesOrderWithCurrentPrice(s.coin1, openedOrder1)
	}

	openedOrder2, _ := s.TransactionRepo.FindOpenedTransactionByCoin(s.tradingStrategy, s.coin2.Id)
	var closedOrder2 *domains.Transaction
	if openedOrder2 != nil {
		closedOrder2 = s.OrderManagerService.CloseFuturesOrderWithCurrentPrice(s.coin2, openedOrder2)
	}
	return closedOrder1, closedOrder2
}

func (s *PairArbitrageStrategyTradingService) hasOpenedOrders() bool {
	openedOrder1, _ := s.TransactionRepo.FindOpenedTransactionByCoin(s.tradingStrategy, s.coin1.Id)
	openedOrder2, _ := s.TransactionRepo.FindOpenedTransactionByCoin(s.tradingStrategy, s.coin2.Id)

	return openedOrder1 != nil || openedOrder2 != nil
}

func (s *PairArbitrageStrategyTradingService) openOrder(coin *domains.Coin, futuresType futureType.FuturesType) {
	stopLossPrice := s.calculateOrderStopLoss(coin, futuresType)
	orderCost := s.calculateCostForOrder()

	zap.S().Debugf("Open order for %v with cost %v", coin.Symbol, orderCost)

	s.OrderManagerService.OpenFuturesOrderWithCostAndFixedStopLossAndTakeProfit(coin, futuresType, orderCost, stopLossPrice, 0)
}

func (s *PairArbitrageStrategyTradingService) calculateOrderStopLoss(coin *domains.Coin, futuresType futureType.FuturesType) float64 {
	if s.stopLossPercent > 0 {
		currentPrice, _ := s.ExchangeDataService.GetCurrentPriceForFutures(coin, s.klineInterval)
		return util.CalculatePriceForStopLoss(currentPrice, s.stopLossPercent, futuresType)
	}

	return float64(0)
}

func (s *PairArbitrageStrategyTradingService) calculateCostForOrder() float64 {
	sumOfProfitByCoin1, _ := s.TransactionRepo.CalculateSumOfProfitByCoin(s.coin1.Id, s.tradingStrategy)
	sumOfProfitByCoin2, _ := s.TransactionRepo.CalculateSumOfProfitByCoin(s.coin2.Id, s.tradingStrategy)

	return util.GetDollarsByCents(((int64(s.startCapitalInCents) + sumOfProfitByCoin1 + sumOfProfitByCoin2) / 2) * int64(s.leverage))
}
