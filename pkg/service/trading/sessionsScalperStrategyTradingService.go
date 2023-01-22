package trading

import (
	"cryptoBot/pkg/constants"
	"cryptoBot/pkg/constants/futureType"
	"cryptoBot/pkg/data/domains"
	"cryptoBot/pkg/repository"
	"cryptoBot/pkg/service/date"
	"cryptoBot/pkg/service/exchange"
	"cryptoBot/pkg/service/indicator"
	"cryptoBot/pkg/service/indicator/techanLib"
	"cryptoBot/pkg/service/orders"
	"cryptoBot/pkg/util"
	"github.com/sdcoffey/techan"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"strconv"
)

//https://youtu.be/ZE0ACEx1U84
var sessionsScalperStrategyTradingService *SessionsScalperStrategyTradingService

func NewSessionsScalperStrategyTradingService(
	transactionRepo repository.Transaction,
	clock date.Clock,
	exchangeDataService *exchange.DataService,
	klineRepo repository.Kline,
	klinesFetcherService *exchange.KlinesFetcherService,
	orderManagerService *orders.OrderManagerService,
	techanConvertorService *techanLib.TechanConvertorService,
	stochasticService *indicator.StochasticService,
	smaTubeService *indicator.SmaTubeService,
	localExtremumTrendService *indicator.LocalExtremumTrendService,
	sessionsService *indicator.SessionsService,
	klineInterval int,
) *SessionsScalperStrategyTradingService {
	if sessionsScalperStrategyTradingService != nil {
		panic("Unexpected try to create second service instance")
	}
	sessionsScalperStrategyTradingService = &SessionsScalperStrategyTradingService{
		KlineRepo:                 klineRepo,
		TransactionRepo:           transactionRepo,
		Clock:                     clock,
		ExchangeDataService:       exchangeDataService,
		KlinesFetcherService:      klinesFetcherService,
		OrderManagerService:       orderManagerService,
		TechanConvertorService:    techanConvertorService,
		StochasticService:         stochasticService,
		SmaTubeService:            smaTubeService,
		LocalExtremumTrendService: localExtremumTrendService,
		SessionsService:           sessionsService,
		klineInterval:             klineInterval,
		periodK:                   5,
		smoothK:                   5,
		periodD:                   5,
		waitingCrossingFastSMA:    false,
		fastSmaLength:             50,
		slowSmaLength:             150,
		takeProfitRatio:           1.5,
		costOfOrderInCents:        100 * 100,
		tradingStrategy:           constants.SESSION_SCALPER,
	}
	return sessionsScalperStrategyTradingService
}

type SessionsScalperStrategyTradingService struct {
	TransactionRepo           repository.Transaction
	KlineRepo                 repository.Kline
	Clock                     date.Clock
	ExchangeDataService       *exchange.DataService
	KlinesFetcherService      *exchange.KlinesFetcherService
	OrderManagerService       *orders.OrderManagerService
	TechanConvertorService    *techanLib.TechanConvertorService
	StochasticService         *indicator.StochasticService
	SmaTubeService            *indicator.SmaTubeService
	LocalExtremumTrendService *indicator.LocalExtremumTrendService
	SessionsService           *indicator.SessionsService
	klineInterval             int
	periodK                   int
	smoothK                   int
	periodD                   int
	waitingCrossingFastSMA    bool
	fastSmaLength             int
	slowSmaLength             int
	takeProfitRatio           float64
	costOfOrderInCents        int
	tradingStrategy           constants.TradingStrategy
}

func (s *SessionsScalperStrategyTradingService) InitializeTrading(coin *domains.Coin) error {
	err := s.OrderManagerService.SetFuturesLeverage(coin, viper.GetInt("strategy.sessionsScalper.futures.leverage"))
	if err != nil {
		return err
	}

	s.KlinesFetcherService.FetchActualKlines(coin, s.klineInterval)

	return nil
}

func (s *SessionsScalperStrategyTradingService) BotAction(coin *domains.Coin) {
	s.KlinesFetcherService.FetchActualKlines(coin, s.klineInterval)

	s.closeOrderIfNeeded(coin)
	openedOrder, _ := s.TransactionRepo.FindOpenedTransaction(s.tradingStrategy)
	if openedOrder != nil {
		return
	}

	if !s.SessionsService.IsSuitableSessionNow() {
		return
	}

	klinesToFetchSize := s.slowSmaLength + 50
	series := s.TechanConvertorService.BuildTimeSeriesByKlines(coin, strconv.Itoa(s.klineInterval), int64(klinesToFetchSize))
	fastSMA := techan.NewSimpleMovingAverage(techan.NewClosePriceIndicator(series), s.fastSmaLength)
	slowSMA := techan.NewSimpleMovingAverage(techan.NewClosePriceIndicator(series), s.slowSmaLength)

	stochasticSignal, stochasticFuturesTypeSignal := s.StochasticService.CalculateStochasticSignal(coin, strconv.Itoa(s.klineInterval), s.periodK, s.smoothK, s.periodD)
	crossFastSmaByTrendSignal, crossFastSmaFuturesTypeSignal := s.SmaTubeService.CrossTheFastSmaByTrendSignal(series, fastSMA, slowSMA)

	isFastSmaBelow := fastSMA.Calculate(klinesToFetchSize - 1).LT(slowSMA.Calculate(klinesToFetchSize - 1))

	if stochasticSignal && crossFastSmaByTrendSignal && stochasticFuturesTypeSignal == crossFastSmaFuturesTypeSignal ||
		s.waitingCrossingFastSMA && crossFastSmaByTrendSignal {
		s.waitingCrossingFastSMA = false

		if isFastSmaBelow && !s.LocalExtremumTrendService.IsTrendDown(coin, strconv.Itoa(s.klineInterval)) {
			zap.S().Infof("Skip open order because Local extremum is not in down trend [%v]", s.Clock.NowTime().Format(constants.DATE_TIME_FORMAT))
			return
		} else if !isFastSmaBelow && !s.LocalExtremumTrendService.IsTrendUp(coin, strconv.Itoa(s.klineInterval)) {
			zap.S().Infof("Skip open order because Local extremum is not in up trend [%v]   [%v]", isFastSmaBelow, s.Clock.NowTime().Format(constants.DATE_TIME_FORMAT))
			return
		}

		zap.S().Infof("stochasticSignal [%v] stochasticFuturesTypeSignal %s   [%v]", stochasticSignal, futureType.GetString(stochasticFuturesTypeSignal), s.Clock.NowTime().Format(constants.DATE_TIME_FORMAT))
		zap.S().Infof("crossFastSmaByTrendSignal [%v] crossFastSmaFuturesTypeSignal %s  [%v]", crossFastSmaByTrendSignal, futureType.GetString(crossFastSmaFuturesTypeSignal), s.Clock.NowTime().Format(constants.DATE_TIME_FORMAT))
		zap.S().Infof("isFastSmaBelow [%v]   [%v]", isFastSmaBelow, s.Clock.NowTime().Format(constants.DATE_TIME_FORMAT))

		s.openOrder(coin, crossFastSmaFuturesTypeSignal)

		return
	}

	if stochasticSignal &&
		(isFastSmaBelow && stochasticFuturesTypeSignal == futureType.SHORT || !isFastSmaBelow && stochasticFuturesTypeSignal == futureType.LONG) &&
		s.SmaTubeService.IsLastKlineClosedInTube(series, fastSMA, slowSMA) {
		s.waitingCrossingFastSMA = true
		zap.S().Infof("Save waitingCrossingFastSMA [%v]  [%v]", s.waitingCrossingFastSMA, s.Clock.NowTime().Format(constants.DATE_TIME_FORMAT))
		return
	}

	if s.waitingCrossingFastSMA && s.SmaTubeService.HasLastKlineGotOutFromTube(series, fastSMA, slowSMA) {
		s.waitingCrossingFastSMA = false
		zap.S().Infof("Delete waitingCrossingFastSMA [%v]  [%v]", s.waitingCrossingFastSMA, s.Clock.NowTime().Format(constants.DATE_TIME_FORMAT))
		return
	}
}

func (s *SessionsScalperStrategyTradingService) closeOrderIfNeeded(coin *domains.Coin) {
	openedOrder, _ := s.TransactionRepo.FindOpenedTransaction(s.tradingStrategy)
	if openedOrder != nil {
		s.OrderManagerService.CloseOrderByFixedStopLossOrTakeProfit(coin, openedOrder, strconv.Itoa(s.klineInterval))
	}
}

func (s *SessionsScalperStrategyTradingService) openOrder(coin *domains.Coin, stochasticFuturesTypeSignal futureType.FuturesType) {
	stopLoss := s.LocalExtremumTrendService.CalculateStopLoss(coin, strconv.Itoa(s.klineInterval), stochasticFuturesTypeSignal)
	currentPrice, _ := s.ExchangeDataService.GetCurrentPrice(coin)
	takeProfit := util.CalculateProfitByRation(currentPrice, stopLoss, stochasticFuturesTypeSignal, s.takeProfitRatio)

	s.OrderManagerService.OpenFuturesOrderWithCostAndFixedStopLossAndTakeProfit(coin, stochasticFuturesTypeSignal, int64(s.costOfOrderInCents), stopLoss, takeProfit)
}
