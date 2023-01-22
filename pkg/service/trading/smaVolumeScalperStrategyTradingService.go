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
	"strconv"
)

//https://youtu.be/ZE0ACEx1U84
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
		takeProfitRatio:                1.5,
		costOfOrderInCents:             100 * 100,
		tradingStrategy:                constants.SMA_VOLUME_SCALPER,
		sma21Length:                    21,
		sma50Length:                    50,
		sma100Length:                   100,
		sma200Length:                   200,
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
	if openedOrder != nil {
		return
	}

	klinesToFetchSize := s.sma200Length + 50
	lastKlineIndex := klinesToFetchSize - 1

	series := s.TechanConvertorService.BuildTimeSeriesByKlines(coin, strconv.Itoa(s.klineInterval), int64(klinesToFetchSize))
	sma21 := techan.NewSimpleMovingAverage(techan.NewClosePriceIndicator(series), s.sma21Length)
	sma50 := techan.NewSimpleMovingAverage(techan.NewClosePriceIndicator(series), s.sma50Length)
	sma100 := techan.NewSimpleMovingAverage(techan.NewClosePriceIndicator(series), s.sma100Length)
	sma200 := techan.NewSimpleMovingAverage(techan.NewClosePriceIndicator(series), s.sma200Length)

	isBullTrend := sma21.Calculate(lastKlineIndex).GT(sma50.Calculate(lastKlineIndex)) &&
		sma50.Calculate(lastKlineIndex).GT(sma100.Calculate(lastKlineIndex)) &&
		sma100.Calculate(lastKlineIndex).GT(sma200.Calculate(lastKlineIndex))

	isBearTrend := sma21.Calculate(lastKlineIndex).LT(sma50.Calculate(lastKlineIndex)) &&
		sma50.Calculate(lastKlineIndex).LT(sma100.Calculate(lastKlineIndex)) &&
		sma100.Calculate(lastKlineIndex).LT(sma200.Calculate(lastKlineIndex))

	if !isBearTrend && !isBullTrend {
		return
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

	volumeSignal := s.RelativeVolumeIndicatorService.CalculateRelativeVolumeSignal(series)

	if !volumeSignal {
		return
	}

	s.openOrder(coin, futureType.GetTypeByBool(bullSignal))
}

func (s *SmaVolumeScalperStrategyTradingService) closeOrderIfNeeded(coin *domains.Coin) {
	openedOrder, _ := s.TransactionRepo.FindOpenedTransaction(s.tradingStrategy)
	if openedOrder != nil {
		s.OrderManagerService.CloseOrderByFixedStopLossOrTakeProfit(coin, openedOrder, strconv.Itoa(s.klineInterval))
	}
}

func (s *SmaVolumeScalperStrategyTradingService) openOrder(coin *domains.Coin, stochasticFuturesTypeSignal futureType.FuturesType) {
	stopLoss := s.LocalExtremumTrendService.CalculateStopLoss(coin, strconv.Itoa(s.klineInterval), stochasticFuturesTypeSignal)
	currentPrice, _ := s.ExchangeDataService.GetCurrentPrice(coin)
	takeProfit := util.CalculateProfitByRation(currentPrice, stopLoss, stochasticFuturesTypeSignal, s.takeProfitRatio)

	s.OrderManagerService.OpenFuturesOrderWithCostAndFixedStopLossAndTakeProfit(coin, stochasticFuturesTypeSignal, int64(s.costOfOrderInCents), stopLoss, takeProfit)
}
