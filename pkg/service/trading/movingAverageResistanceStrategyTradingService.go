package trading

import (
	"cryptoBot/configs"
	"cryptoBot/pkg/api"
	"cryptoBot/pkg/constants"
	"cryptoBot/pkg/constants/futureType"
	"cryptoBot/pkg/data/domains"
	"cryptoBot/pkg/repository"
	"cryptoBot/pkg/service/date"
	"cryptoBot/pkg/service/exchange"
	"cryptoBot/pkg/service/indicator"
	"cryptoBot/pkg/service/orders"
	"cryptoBot/pkg/util"
	"database/sql"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

var movingAverageResistanceStrategyTradingServiceImpl *MovingAverageResistanceStrategyTradingService

func NewMovingAverageResistanceStrategyTradingService(transactionRepo repository.Transaction, priceChangeRepo repository.PriceChange,
	exchangeApi api.ExchangeApi, clock date.Clock, exchangeDataService *exchange.DataService, klineRepo repository.Kline,
	priceChangeTrackingService *orders.PriceChangeTrackingService, movingAverageService *indicator.MovingAverageService) *MovingAverageResistanceStrategyTradingService {
	if movingAverageResistanceStrategyTradingServiceImpl != nil {
		panic("Unexpected try to create second service instance")
	}
	movingAverageResistanceStrategyTradingServiceImpl = &MovingAverageResistanceStrategyTradingService{
		klineRepo:                  klineRepo,
		transactionRepo:            transactionRepo,
		priceChangeRepo:            priceChangeRepo,
		exchangeApi:                exchangeApi,
		Clock:                      clock,
		ExchangeDataService:        exchangeDataService,
		PriceChangeTrackingService: priceChangeTrackingService,
		MovingAverageService:       movingAverageService,
	}
	return movingAverageResistanceStrategyTradingServiceImpl
}

type MovingAverageResistanceStrategyTradingService struct {
	transactionRepo            repository.Transaction
	priceChangeRepo            repository.PriceChange
	klineRepo                  repository.Kline
	exchangeApi                api.ExchangeApi
	Clock                      date.Clock
	ExchangeDataService        *exchange.DataService
	PriceChangeTrackingService *orders.PriceChangeTrackingService
	MovingAverageService       *indicator.MovingAverageService
}

func (s *MovingAverageResistanceStrategyTradingService) InitializeTrading(coin *domains.Coin) error {
	return nil
}

func (s *MovingAverageResistanceStrategyTradingService) BotAction(coin *domains.Coin) {
	if !configs.RuntimeConfig.TradingEnabled {
		return
	}

	//todo fetch needed bars from bybit

	s.BotSingleAction(coin)
}

func (s *MovingAverageResistanceStrategyTradingService) BotSingleAction(coin *domains.Coin) {
	s.closeOrderIfProfitEnough(coin)

	if s.Clock.NowTime().Minute()%viper.GetInt("strategy.maResistance.interval") == 0 {
		s.calculateMovingAverage(coin)
	}
}

func (s *MovingAverageResistanceStrategyTradingService) calculateMovingAverage(coin *domains.Coin) {
	openedOrder, _ := s.transactionRepo.FindOpenedTransaction(constants.MOVING_AVARAGE_RESISTANCE)
	if openedOrder != nil {
		return
	}

	shortAvgs := s.MovingAverageService.CalculateAvg(coin, viper.GetInt("strategy.maResistance.length.short"), viper.GetInt("strategy.maResistance.length.short"))
	mediumAvgs := s.MovingAverageService.CalculateAvg(coin, viper.GetInt("strategy.maResistance.length.medium"), viper.GetInt("strategy.maResistance.length.short"))

	if shortAvgs == nil || mediumAvgs == nil {
		zap.S().Errorf("Can't calculate direction of moving averages")
		return
	}

	klines, err := s.klineRepo.FindAllByCoinIdAndIntervalAndCloseTimeLessOrderByOpenTimeWithLimit(coin.Id, viper.GetString("strategy.maResistance.interval"), s.Clock.NowTime(), 3)
	if err != nil {
		zap.S().Errorf("Error during FindClosedAtMoment at %v: %s", s.Clock.NowTime(), err.Error())
		return
	}

	lastKline := klines[len(klines)-1]
	lastShortMa := shortAvgs[len(shortAvgs)-1]

	if s.isUpTrend(shortAvgs, mediumAvgs) {
		isLastKlineCrossUpTube := lastKline.Open < lastShortMa && lastKline.Close > lastShortMa
		if isLastKlineCrossUpTube && s.isAllKlinesInTubeMa(mediumAvgs, shortAvgs, klines[0:len(klines)-1]) {
			zap.S().Infof("Open LONG")
			s.openOrder(coin, futureType.LONG)
		}
	}

	if s.isDownTrend(shortAvgs, mediumAvgs) {
		isLasKlineCrossDownTube := lastKline.Open > lastShortMa && lastKline.Close < lastShortMa
		if isLasKlineCrossDownTube && s.isAllKlinesInTubeMa(shortAvgs, mediumAvgs, klines[0:len(klines)-1]) {
			zap.S().Infof("Open SHORT")
			s.openOrder(coin, futureType.SHORT)
		}
	}
}

func (s *MovingAverageResistanceStrategyTradingService) isAllKlinesInTubeMa(downMA []int64, upMA []int64, klines []*domains.Kline) bool {
	for i := len(klines) - 1; i >= 0; i-- {
		if klines[i].Open < downMA[i] || klines[i].Close < downMA[i] ||
			klines[i].Open > upMA[i] || klines[i].Close > upMA[i] {
			return false
		}
	}

	return true
}

func (s *MovingAverageResistanceStrategyTradingService) isUpTrend(shortAvgs []int64, mediumAvgs []int64) bool {
	for i := 0; i < len(shortAvgs); i++ {
		if mediumAvgs[i] > shortAvgs[i] {
			return false
		}
	}
	return true
}

func (s *MovingAverageResistanceStrategyTradingService) isDownTrend(shortAvgs []int64, mediumAvgs []int64) bool {
	for i := 0; i < len(shortAvgs); i++ {
		if mediumAvgs[i] < shortAvgs[i] {
			return false
		}
	}
	return true
}

//copied
func (s *MovingAverageResistanceStrategyTradingService) openOrder(coin *domains.Coin, futuresType futureType.FuturesType) {
	currentPrice, err := s.ExchangeDataService.GetCurrentPrice(coin)
	if err != nil {
		zap.S().Errorf("Error during GetCurrentCoinPrice at %v: %s", s.Clock.NowTime(), err.Error())
		return
	}
	amountTransaction := util.CalculateAmountByPriceAndCost(currentPrice, viper.GetInt64("strategy.ma.cost"))
	orderDto, err2 := s.exchangeApi.OpenFuturesOrder(coin, amountTransaction, currentPrice, futuresType, 10)
	if err2 != nil {
		zap.S().Errorf("Error during OpenFuturesOrder: %s", err2.Error())
		return
	}

	transaction := s.createOpenTransactionByOrderResponseDto(coin, futuresType, orderDto)
	if err3 := s.transactionRepo.SaveTransaction(&transaction); err3 != nil {
		zap.S().Errorf("Error during SaveTransaction: %s", err3.Error())
		return
	}

	zap.S().Infof("at %v Order opened  with price %v and type [%v] (0-L, 1-S)", s.Clock.NowTime(), currentPrice, futuresType)
}

//copied
func (s *MovingAverageResistanceStrategyTradingService) createOpenTransactionByOrderResponseDto(coin *domains.Coin, futuresType futureType.FuturesType,
	orderDto api.OrderResponseDto) domains.Transaction {
	transaction := domains.Transaction{
		TradingStrategy: constants.MOVING_AVARAGE_RESISTANCE,
		FuturesType:     futuresType,
		CoinId:          coin.Id,
		Amount:          orderDto.GetAmount(),
		Price:           orderDto.CalculateAvgPrice(),
		TotalCost:       orderDto.CalculateTotalCost(),
		Commission:      orderDto.CalculateCommissionInUsd(),
		CreatedAt:       s.Clock.NowTime(),
	}

	if futuresType == futureType.LONG {
		transaction.TransactionType = constants.BUY
	} else {
		transaction.TransactionType = constants.SELL
	}
	return transaction
}

//copied
func (s *MovingAverageResistanceStrategyTradingService) closeOrder(openTransaction *domains.Transaction, coin *domains.Coin) {
	currentPrice, err := s.ExchangeDataService.GetCurrentPrice(coin)
	if err != nil {
		zap.S().Errorf("Error during GetCurrentCoinPrice at %v: %s", s.Clock.NowTime(), err.Error())
		return
	}

	orderResponseDto, err := s.exchangeApi.CloseFuturesOrder(coin, openTransaction, currentPrice)
	if err != nil {
		zap.S().Errorf("Error during CloseFuturesOrder: %s", err.Error())
		return
	}

	closeTransaction := s.createCloseTransactionByOrderResponseDto(coin, openTransaction, orderResponseDto)
	if errT := s.transactionRepo.SaveTransaction(&closeTransaction); errT != nil {
		zap.S().Errorf("Error during SaveTransaction: %s", errT.Error())
		return
	}

	openTransaction.RelatedTransactionId = sql.NullInt64{Int64: closeTransaction.Id, Valid: true}
	_ = s.transactionRepo.SaveTransaction(openTransaction)
}

//copied
func (s *MovingAverageResistanceStrategyTradingService) createCloseTransactionByOrderResponseDto(coin *domains.Coin, openedTransaction *domains.Transaction,
	orderDto api.OrderResponseDto) domains.Transaction {

	var buyCost int64
	var sellCost int64
	var transactionType constants.TransactionType

	if openedTransaction.FuturesType == futureType.LONG {
		buyCost = openedTransaction.TotalCost
		sellCost = orderDto.CalculateTotalCost()
		transactionType = constants.SELL
	} else {
		buyCost = orderDto.CalculateTotalCost()
		sellCost = openedTransaction.TotalCost
		transactionType = constants.BUY
	}

	profitInUsd := sellCost - buyCost - orderDto.CalculateCommissionInUsd() - openedTransaction.Commission

	transaction := domains.Transaction{
		TradingStrategy:      constants.MOVING_AVARAGE_RESISTANCE,
		FuturesType:          openedTransaction.FuturesType,
		TransactionType:      transactionType,
		CoinId:               coin.Id,
		Amount:               orderDto.GetAmount(),
		Price:                orderDto.CalculateAvgPrice(),
		TotalCost:            orderDto.CalculateTotalCost(),
		Commission:           orderDto.CalculateCommissionInUsd(),
		RelatedTransactionId: sql.NullInt64{Int64: openedTransaction.Id, Valid: true},
		Profit:               sql.NullInt64{Int64: profitInUsd, Valid: true},
		PercentProfit:        sql.NullFloat64{Float64: float64(profitInUsd) / float64(openedTransaction.TotalCost) * 100, Valid: true},
		CreatedAt:            s.Clock.NowTime(),
	}
	return transaction
}

func (s *MovingAverageResistanceStrategyTradingService) closeOrderIfProfitEnough(coin *domains.Coin) {
	openedOrder, _ := s.transactionRepo.FindOpenedTransaction(constants.MOVING_AVARAGE_RESISTANCE)

	if openedOrder == nil {
		return
	}

	if s.shouldCloseByStopLoss(openedOrder, coin) {
		s.closeOrder(openedOrder, coin)
		return
	}
	if s.shouldCloseWithProfit(openedOrder, coin) {
		s.closeOrder(openedOrder, coin)
		return
	}
	//if s.isCloseToBreakeven(openedOrder, coin) {
	//	s.closeOrder(openedOrder, coin)
	//	return
	//}
	//if s.isProfitByTrolling(openedOrder, coin) {
	//	s.closeOrder(openedOrder, coin)
	//	return
	//}
	//if s.isCurrentPriceIntersectMA(openedOrder, coin) {
	//	s.closeOrder(openedOrder, coin)
	//	return
	//}
}

func (s *MovingAverageResistanceStrategyTradingService) shouldCloseByStopLoss(lastTransaction *domains.Transaction, coin *domains.Coin) bool {
	if s.Clock.NowTime().Minute()%viper.GetInt("strategy.maResistance.interval") != 0 {
		return false
	}
	currentPrice, err := s.ExchangeDataService.GetCurrentPrice(coin)
	if err != nil {
		zap.S().Errorf("Error during GetCurrentCoinPrice at %v: %s", s.Clock.NowTime(), err.Error())
		return false
	}

	if lastTransaction.FuturesType == futureType.LONG {
		orderProfitInPercent := util.CalculateChangeInPercents(lastTransaction.Price, currentPrice)
		if orderProfitInPercent <= viper.GetFloat64("strategy.maResistance.percentStopLoss") {
			zap.S().Infof("at %v close order by stop loss price=%v currentProfitInPercent=%v", s.Clock.NowTime(), currentPrice, orderProfitInPercent)
			return true
		}
	}

	if lastTransaction.FuturesType == futureType.SHORT {
		orderProfitInPercent := -1 * util.CalculateChangeInPercents(lastTransaction.Price, currentPrice)
		if orderProfitInPercent <= viper.GetFloat64("strategy.maResistance.percentStopLoss") {
			zap.S().Infof("at %v close order by stop loss price=%v currentProfitInPercent=%v", s.Clock.NowTime(), currentPrice, orderProfitInPercent)
			return true
		}
	}

	return false
}

func (s *MovingAverageResistanceStrategyTradingService) shouldCloseWithProfit(lastTransaction *domains.Transaction, coin *domains.Coin) bool {
	currentPrice, err := s.ExchangeDataService.GetCurrentPrice(coin)
	if err != nil {
		zap.S().Errorf("Error during GetCurrentCoinPrice at %v: %s", s.Clock.NowTime(), err.Error())
		return false
	}

	if lastTransaction.FuturesType == futureType.LONG {
		orderProfitInPercent := util.CalculateChangeInPercents(lastTransaction.Price, currentPrice)
		if orderProfitInPercent >= viper.GetFloat64("strategy.maResistance.percentProfit") {
			zap.S().Infof("At %v close LONG with profit price=%v currentProfitInPercent=%v", s.Clock.NowTime(), currentPrice, orderProfitInPercent)
			return true
		}
	}

	if lastTransaction.FuturesType == futureType.SHORT {
		orderProfitInPercent := -1 * util.CalculateChangeInPercents(lastTransaction.Price, currentPrice)
		if orderProfitInPercent >= viper.GetFloat64("strategy.maResistance.percentProfit") {
			zap.S().Infof("At %v close SHORT with profit price=%v currentProfitInPercent=%v", s.Clock.NowTime(), currentPrice, orderProfitInPercent)
			return true
		}
	}

	return false
}

func (s *MovingAverageResistanceStrategyTradingService) isCurrentPriceIntersectMA(lastTransaction *domains.Transaction, coin *domains.Coin) bool {
	klines, err := s.klineRepo.FindAllByCoinIdAndIntervalAndCloseTimeLessOrderByOpenTimeWithLimit(coin.Id, viper.GetString("strategy.maResistance.interval"), s.Clock.NowTime(), 1)
	if err != nil {
		zap.S().Errorf("Error during FindClosedAtMoment at %v: %s", s.Clock.NowTime(), err.Error())
		return false
	}

	lastKline := klines[0]

	movingAvgs := s.MovingAverageService.CalculateAvg(coin, viper.GetInt("strategy.maResistance.length.medium"), 1)
	percentsProfit := util.CalculateChangeInPercents(lastTransaction.Price, lastKline.Close)

	if lastTransaction.FuturesType == futureType.LONG {
		if lastKline.Close < movingAvgs[len(movingAvgs)-1] {
			zap.S().Infof("At %v close intersect MA LONG below MA price=%v movingAvgs=%v profit=%v \n", s.Clock.NowTime(), lastKline.Close, movingAvgs, percentsProfit)
			return true
		}
	}

	if lastTransaction.FuturesType == futureType.SHORT {
		if lastKline.Close > movingAvgs[len(movingAvgs)-1] {
			zap.S().Infof("At %v close intersect MA SHORT MA price=%v movingAvgs=%v profit=%v \n", s.Clock.NowTime(), lastKline.Close, movingAvgs, percentsProfit)
			zap.S().Infof("At %v close intersect MA SHORT MA price=%v movingAvgs=%v profit=%v \n", s.Clock.NowTime(), lastKline.Close, movingAvgs, percentsProfit)
			return true
		}
	}

	return false
}

func (s *MovingAverageResistanceStrategyTradingService) isProfitByTrolling(lastTransaction *domains.Transaction, coin *domains.Coin) bool {
	currentPrice, err := s.ExchangeDataService.GetCurrentPrice(coin)
	if err != nil {
		zap.S().Errorf("Error during GetCurrentCoinPrice at %v: %s", s.Clock.NowTime(), err.Error())
		return false
	}

	priceChange := s.PriceChangeTrackingService.GetChangePrice(lastTransaction.Id, currentPrice)

	if lastTransaction.FuturesType == futureType.LONG {
		// close order if price on percentProfit lower from high
		priceChangeInPercent := util.CalculateChangeInPercents(priceChange.HighPrice, currentPrice)
		if priceChangeInPercent < -1*viper.GetFloat64("strategy.maResistance.percentTrollingProfit") {
			zap.S().Infof("At %v close order trolling. Higher price %v current price %v percent %v",
				s.Clock.NowTime(), priceChange.HighPrice, currentPrice, priceChangeInPercent)
			return true
		}
	}
	if lastTransaction.FuturesType == futureType.SHORT {
		// close order if price on percentProfit higher from low
		priceChangeInPercent := util.CalculateChangeInPercents(priceChange.LowPrice, currentPrice)
		if priceChangeInPercent > viper.GetFloat64("strategy.maResistance.percentTrollingProfit") {
			zap.S().Infof("At %v close order trolling. Lower price %v current price %v percent %v",
				s.Clock.NowTime(), priceChange.LowPrice, currentPrice, priceChangeInPercent)
			return true
		}
	}

	return false
}
