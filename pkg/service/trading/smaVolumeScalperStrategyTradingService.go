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
	"math"
	"strconv"
)

// https://youtu.be/ZE0ACEx1U84
var smaVolumeScalperStrategyTradingService *SmaVolumeScalperStrategyTradingService

func NewSmaVolumeScalperStrategyTradingService(
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
	relativeVolumeIndicatorService *indicator.RelativeVolumeIndicatorService,
	klineInterval int,
) *SmaVolumeScalperStrategyTradingService {
	if smaVolumeScalperStrategyTradingService != nil {
		panic("Unexpected try to create second service instance")
	}
	smaVolumeScalperStrategyTradingService = &SmaVolumeScalperStrategyTradingService{
		KlineRepo:                      klineRepo,
		TransactionRepo:                transactionRepo,
		Clock:                          clock,
		ExchangeDataService:            exchangeDataService,
		KlinesFetcherService:           klinesFetcherService,
		OrderManagerService:            orderManagerService,
		TechanConvertorService:         techanConvertorService,
		StochasticService:              stochasticService,
		SmaTubeService:                 smaTubeService,
		LocalExtremumTrendService:      localExtremumTrendService,
		RelativeVolumeIndicatorService: relativeVolumeIndicatorService,
		klineInterval:                  klineInterval,
		periodK:                        5,
		smoothK:                        5,
		periodD:                        5,
		waitingCrossingFastSMA:         false,
		fastSmaLength:                  50,
		slowSmaLength:                  150,
		takeProfitRatio:                0.35,
		costOfOrderInCents:             100 * 100,
		tradingStrategy:                constants.SMA_VOLUME_SCALPER,
		sma21Length:                    21,
		sma50Length:                    50,
		sma100Length:                   100,
		sma200Length:                   200,

		BULL_STATUS:       false,
		BEAR_STATUS:       false,
		BULL_TREND_STATUS: false,
		BEAR_TREND_STATUS: false,
	}
	return smaVolumeScalperStrategyTradingService
}

type SmaVolumeScalperStrategyTradingService struct {
	TransactionRepo                repository.Transaction
	KlineRepo                      repository.Kline
	Clock                          date.Clock
	ExchangeDataService            *exchange.DataService
	KlinesFetcherService           *exchange.KlinesFetcherService
	OrderManagerService            *orders.OrderManagerService
	TechanConvertorService         *techanLib.TechanConvertorService
	StochasticService              *indicator.StochasticService
	SmaTubeService                 *indicator.SmaTubeService
	LocalExtremumTrendService      *indicator.LocalExtremumTrendService
	RelativeVolumeIndicatorService *indicator.RelativeVolumeIndicatorService
	klineInterval                  int
	periodK                        int
	smoothK                        int
	periodD                        int
	waitingCrossingFastSMA         bool
	fastSmaLength                  int
	sma21Length                    int
	sma50Length                    int
	sma100Length                   int
	sma200Length                   int
	slowSmaLength                  int
	takeProfitRatio                float64
	costOfOrderInCents             int
	tradingStrategy                constants.TradingStrategy

	BULL_STATUS       bool
	BEAR_STATUS       bool
	BULL_TREND_STATUS bool
	BEAR_TREND_STATUS bool
}

func (s *SmaVolumeScalperStrategyTradingService) InitializeTrading(coin *domains.Coin) error {
	err := s.OrderManagerService.SetFuturesLeverage(coin, viper.GetInt("strategy.smaVolumeScalper.futures.leverage"))
	if err != nil {
		return err
	}

	s.KlinesFetcherService.FetchActualKlines(coin, s.klineInterval)

	return nil
}

func (s *SmaVolumeScalperStrategyTradingService) BotAction(coin *domains.Coin) {
	s.KlinesFetcherService.FetchActualKlines(coin, s.klineInterval)

	s.closeOrderIfNeeded(coin)
	openedOrder, _ := s.TransactionRepo.FindOpenedTransaction(s.tradingStrategy)
	//if openedOrder != nil {
	//	return
	//}

	////TODO test closing with logic below:
	if openedOrder != nil {
		currentProfit, _ := s.OrderManagerService.CalculateCurrentProfitInPercentWithoutLeverage(coin, openedOrder)
		minProfitInPercent := 0.46
		if currentProfit > minProfitInPercent {
			s.OrderManagerService.CloseFuturesOrderWithCurrentPrice(coin, openedOrder)
		} else {
			return
		}
	}

	klinesToFetchSize := s.sma200Length * 2

	series := s.TechanConvertorService.BuildTimeSeriesByKlines(coin, strconv.Itoa(s.klineInterval), int64(klinesToFetchSize))

	lastKlineIndex := series.LastIndex()

	smma21 := techan.NewEMAIndicator(techan.NewClosePriceIndicator(series), s.sma21Length)
	smma50 := techan.NewEMAIndicator(techan.NewClosePriceIndicator(series), s.sma50Length)
	smma100 := techan.NewEMAIndicator(techan.NewClosePriceIndicator(series), s.sma100Length)
	smma200 := techan.NewEMAIndicator(techan.NewClosePriceIndicator(series), s.sma200Length)

	smmma21Last := smma21.Calculate(lastKlineIndex)
	smma50Last := smma50.Calculate(lastKlineIndex)
	smma100Last := smma100.Calculate(lastKlineIndex)
	smma200Last := smma200.Calculate(lastKlineIndex)

	//zap.S().Infof("at %v 21=%v 50=%v 100=%v 200=%v", s.Clock.NowTime().Format(constants.DATE_TIME_FORMAT),
	//	smmma21Last.FormattedString(0), smma50Last.FormattedString(0), smma100Last.FormattedString(0), smma200Last.FormattedString(0))

	smaDifferenceInPercent := 0.5
	isBullTrend := smmma21Last.GT(smma50Last) && util.CalculateChangeInPercentsAbsBig(smmma21Last, smma50Last) > smaDifferenceInPercent &&
		smma50Last.GT(smma100Last) && util.CalculateChangeInPercentsAbsBig(smma50Last, smma100Last) > smaDifferenceInPercent &&
		smma100Last.GT(smma200Last) && util.CalculateChangeInPercentsAbsBig(smma100Last, smma200Last) > smaDifferenceInPercent

	isBearTrend := smmma21Last.LT(smma50Last) && util.CalculateChangeInPercentsAbsBig(smmma21Last, smma50Last) > smaDifferenceInPercent &&
		smma50Last.LT(smma100Last) && util.CalculateChangeInPercentsAbsBig(smma50Last, smma100Last) > smaDifferenceInPercent &&
		smma100Last.LT(smma200Last) && util.CalculateChangeInPercentsAbsBig(smma100Last, smma200Last) > smaDifferenceInPercent

	if !isBearTrend && !isBullTrend {
		if s.BEAR_TREND_STATUS {
			zap.S().Infof("END BEAR TREND at %v", s.Clock.NowTime().Format(constants.DATE_TIME_FORMAT))
			s.BEAR_TREND_STATUS = false
		}

		if s.BULL_TREND_STATUS {
			zap.S().Infof("END BULL TREND at %v", s.Clock.NowTime().Format(constants.DATE_TIME_FORMAT))
			s.BULL_TREND_STATUS = false
		}

		//if openedOrder == nil {
		return
		//}
	}

	if !s.BEAR_TREND_STATUS && isBearTrend {
		zap.S().Infof("START BEAR TREND at %v", s.Clock.NowTime().Format(constants.DATE_TIME_FORMAT))
		s.BEAR_TREND_STATUS = true
	}

	if !s.BULL_TREND_STATUS && isBullTrend {
		zap.S().Infof("START BULL TREND at %v", s.Clock.NowTime().Format(constants.DATE_TIME_FORMAT))
		s.BULL_TREND_STATUS = true
	}

	bearSignal := series.Candles[lastKlineIndex-3].ClosePrice.GT(series.Candles[lastKlineIndex-3].OpenPrice) &&
		series.Candles[lastKlineIndex-2].ClosePrice.GT(series.Candles[lastKlineIndex-2].OpenPrice) &&
		series.Candles[lastKlineIndex-1].ClosePrice.GT(series.Candles[lastKlineIndex-1].OpenPrice) &&
		series.Candles[lastKlineIndex-0].ClosePrice.LT(series.Candles[lastKlineIndex-1].OpenPrice)
	bullSignal := series.Candles[lastKlineIndex-3].ClosePrice.LT(series.Candles[lastKlineIndex-3].OpenPrice) &&
		series.Candles[lastKlineIndex-2].ClosePrice.LT(series.Candles[lastKlineIndex-2].OpenPrice) &&
		series.Candles[lastKlineIndex-1].ClosePrice.LT(series.Candles[lastKlineIndex-1].OpenPrice) &&
		series.Candles[lastKlineIndex-0].ClosePrice.GT(series.Candles[lastKlineIndex-1].OpenPrice)

	if !bearSignal && !bullSignal {
		return
	}

	if bearSignal {
		zap.S().Infof("BEAR SIGNAL at %v", s.Clock.NowTime().Format(constants.DATE_TIME_FORMAT))
	}

	if bullSignal {
		zap.S().Infof("BULL SIGNAL at %v", s.Clock.NowTime().Format(constants.DATE_TIME_FORMAT))
	}

	//if openedOrder != nil {
	//	currentProfit, _ := s.OrderManagerService.CalculateCurrentProfitInPercent(coin, openedOrder)
	//	minProfitInPercent := 0.25
	//	if bearSignal && openedOrder.FuturesType == futureType.LONG && currentProfit > minProfitInPercent {
	//		s.OrderManagerService.CloseFuturesOrderWithCurrentPrice(coin, openedOrder)
	//	}
	//	if bullSignal && openedOrder.FuturesType == futureType.SHORT && currentProfit > minProfitInPercent {
	//		s.OrderManagerService.CloseFuturesOrderWithCurrentPrice(coin, openedOrder)
	//	}
	//	return
	//}

	signalByFloat := s.RelativeVolumeIndicatorService.CalculateRelativeVolumeSignalWithFloats(series)

	zap.S().Infof("volume signal %v at %v", signalByFloat, s.Clock.NowTime().Format(constants.DATE_TIME_FORMAT))

	if !signalByFloat {
		return
	}

	if isBullTrend && bullSignal {
		s.openOrder(coin, futureType.LONG)
	} else if isBearTrend && bearSignal {
		s.openOrder(coin, futureType.SHORT)
	}
}

func (s *SmaVolumeScalperStrategyTradingService) closeOrderIfNeeded(coin *domains.Coin) {
	openedOrder, _ := s.TransactionRepo.FindOpenedTransaction(s.tradingStrategy)
	if openedOrder != nil {
		s.OrderManagerService.CloseOrderByFixedStopLossOrTakeProfit(coin, openedOrder, strconv.Itoa(s.klineInterval))
	}

	openedOrder, _ = s.TransactionRepo.FindOpenedTransaction(s.tradingStrategy)
	if openedOrder != nil && s.OrderManagerService.ShouldCloseByTrailingTakeProfitWithoutLeverage(coin, openedOrder) {
		s.OrderManagerService.CloseFuturesOrderWithCurrentPrice(coin, openedOrder)
	}
}

func (s *SmaVolumeScalperStrategyTradingService) openOrder(coin *domains.Coin, futuresTypeSignal futureType.FuturesType) {
	stopLoss := s.LocalExtremumTrendService.CalculateStopLoss(coin, strconv.Itoa(s.klineInterval), futuresTypeSignal)

	currentPrice, _ := s.ExchangeDataService.GetCurrentPrice(coin)
	//takeProfit := util.CalculateProfitByRation(currentPrice, stopLoss, futuresTypeSignal, s.takeProfitRatio)

	takeProfit := float64(0) //util.CalculatePriceForTakeProfit(currentPrice, 0.86, futuresTypeSignal) //int64(0)

	stopLossInPercent := util.CalculateProfitInPercent(currentPrice, stopLoss, futuresTypeSignal)
	if math.Abs(stopLossInPercent) > 2.5 {
		zap.S().Infof("Skip order with stopLoss %v", stopLossInPercent)
		return
	}

	isNextOrderFake := s.isNextOrderFake(coin)
	costOrOrder := s.calculateCurrentWalletValue(coin)

	s.OrderManagerService.OpenFuturesOrderWithCostAndFixedStopLossAndTakeProfitAndFake(coin, "", futuresTypeSignal, costOrOrder, stopLoss, takeProfit, isNextOrderFake)
}

func (s *SmaVolumeScalperStrategyTradingService) isNextOrderFake(coin *domains.Coin) bool {
	transaction, err := s.TransactionRepo.FindLastByCoinId(coin.Id, s.tradingStrategy)
	if transaction == nil || err != nil {
		return false
	}

	if transaction.Profit.Valid {
		return transaction.Profit.Int64 < 0
	}

	return false
}

func (s *SmaVolumeScalperStrategyTradingService) calculateCurrentWalletValue(coin *domains.Coin) float64 {
	sumOfProfitByCoin, _ := s.TransactionRepo.CalculateSumOfProfitByCoin(coin.Id, s.tradingStrategy)

	return util.GetDollarsByCents((int64(s.costOfOrderInCents) + sumOfProfitByCoin) * viper.GetInt64("strategy.smaVolumeScalper.futures.leverage"))
}
