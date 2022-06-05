package trading

import (
	"cryptoBot/configs"
	"cryptoBot/pkg/api"
	"cryptoBot/pkg/constants"
	"cryptoBot/pkg/data/domains"
	"cryptoBot/pkg/repository"
	"cryptoBot/pkg/service/date"
	"cryptoBot/pkg/service/exchange"
	"cryptoBot/pkg/service/indicator"
	"cryptoBot/pkg/util"
	"database/sql"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

var movingAverageResistanceStrategyTradingServiceImpl *MovingAverageResistanceStrategyTradingService

func NewMovingAverageResistanceStrategyTradingService(transactionRepo repository.Transaction, priceChangeRepo repository.PriceChange,
	exchangeApi api.ExchangeApi, clock date.Clock, exchangeDataService *exchange.DataService, klineRepo repository.Kline,
	priceChangeTrackingService *PriceChangeTrackingService, movingAverageService *indicator.MovingAverageService) *MovingAverageResistanceStrategyTradingService {
	if movingAverageStrategyTradingServiceImpl != nil {
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
	PriceChangeTrackingService *PriceChangeTrackingService
	MovingAverageService       *indicator.MovingAverageService

	isWaitingForCrossUp   bool
	isWaitingForCrossDown bool
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

	s.calculateMovingAverage(coin)
}

func (s *MovingAverageResistanceStrategyTradingService) calculateMovingAverage(coin *domains.Coin) {
	openedOrder, _ := s.transactionRepo.FindOpenedTransaction(constants.MOVING_AVARAGE_RESISTANCE)
	if openedOrder != nil {
		return
	}

	shortAvgs := s.MovingAverageService.CalculateAvg(coin, viper.GetInt("strategy.maResistance.length.short"), 2)
	mediumAvgs := s.MovingAverageService.CalculateAvg(coin, viper.GetInt("strategy.maResistance.length.medium"), 2)

	if shortAvgs == nil || len(shortAvgs) < 2 || mediumAvgs == nil || len(mediumAvgs) < 2 {
		zap.S().Errorf("Can't calculate direction of moving averages")
		return
	}

	klines, err := s.klineRepo.FindAllByCoinIdAndIntervalAndCloseTimeLessOrderByOpenTimeWithLimit(coin.Id, viper.GetString("strategy.maResistance.interval"), s.Clock.NowTime(), 2)
	if err != nil {
		zap.S().Errorf("Error during FindClosedAtMoment at %v: %s", s.Clock.NowTime(), err.Error())
		return
	}

	prevKline := klines[0]
	lastKline := klines[1]

	if shortAvgs[0] > mediumAvgs[0] && shortAvgs[1] > mediumAvgs[1] { //is up trend

		isLastKlineCloseBelowTube := lastKline.Close < mediumAvgs[1]
		if isLastKlineCloseBelowTube {
			s.isWaitingForCrossUp = false
			return
		}

		isPrevKlineAboveMA := prevKline.Open > shortAvgs[0] && prevKline.Close > shortAvgs[0]
		isLastKlineCloseInTube := lastKline.Open > shortAvgs[1] && lastKline.Close < shortAvgs[1] && lastKline.Close > mediumAvgs[1]

		if isPrevKlineAboveMA && isLastKlineCloseInTube {
			s.isWaitingForCrossUp = true
		}
		if lastKline.Open < shortAvgs[1] && lastKline.Close > shortAvgs[1] { // if cross up
			if s.isWaitingForCrossUp {
				zap.S().Infof("Open LONG")
				s.openOrder(coin, constants.LONG)
			}
		}
	}

	if shortAvgs[0] < mediumAvgs[0] && shortAvgs[1] < mediumAvgs[1] { //is down trend

		isLastKlineCloseAboveTube := lastKline.Close > mediumAvgs[1]
		if isLastKlineCloseAboveTube {
			s.isWaitingForCrossUp = false
			return
		}

		isPrevKlineBelowTube := prevKline.Open < shortAvgs[0] && prevKline.Close < shortAvgs[0]
		isLastKlineCloseInTube := lastKline.Open < shortAvgs[1] && lastKline.Close > shortAvgs[1] && lastKline.Close < mediumAvgs[1]

		if isPrevKlineBelowTube && isLastKlineCloseInTube { // if cross up
			s.isWaitingForCrossDown = true
		}
		if lastKline.Open > shortAvgs[1] && lastKline.Close < shortAvgs[1] { // if cross down
			if s.isWaitingForCrossDown {
				zap.S().Infof("Open LONG")
				s.openOrder(coin, constants.SHORT)
			}
		}
	}
}

//copied
func (s *MovingAverageResistanceStrategyTradingService) openOrder(coin *domains.Coin, futuresType constants.FuturesType) {
	currentPrice, err := s.ExchangeDataService.GetCurrentPrice(coin)
	if err != nil {
		zap.S().Errorf("Error during GetCurrentCoinPrice at %v: %s", s.Clock.NowTime(), err.Error())
		return
	}
	amountTransaction := util.CalculateAmountByPriceAndCost(currentPrice, viper.GetInt64("strategy.ma.cost"))
	orderDto, err2 := s.exchangeApi.OpenFuturesOrder(coin, amountTransaction, currentPrice, futuresType, viper.GetInt("strategy.ma.futures.leverage"))
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
func (s *MovingAverageResistanceStrategyTradingService) createOpenTransactionByOrderResponseDto(coin *domains.Coin, futuresType constants.FuturesType,
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

	if futuresType == constants.LONG {
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

	orderResponseDto, err := s.exchangeApi.CloseFuturesOrder(openTransaction, currentPrice)
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

	if openedTransaction.FuturesType == constants.LONG {
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

	//if s.shouldCloseByStopLoss(openedOrder, coin) {
	//s.closeOrder(openedOrder, coin)
	//	return
	//}
	//if s.shouldCloseWithProfit(openedOrder, coin) {
	//	s.closeOrder(openedOrder, coin)
	//	return
	//}
	//if s.isCloseToBreakeven(openedOrder, coin) {
	//	s.closeOrder(openedOrder, coin)
	//	return
	//}
	//if s.isProfitByTrolling(openedOrder, coin) {
	//	s.closeOrder(openedOrder, coin)
	//	return
	//}
	if s.isCurrentPriceIntersectMA(openedOrder, coin) {
		s.closeOrder(openedOrder, coin)
		return
	}
}

func (s *MovingAverageResistanceStrategyTradingService) isCurrentPriceIntersectMA(lastTransaction *domains.Transaction, coin *domains.Coin) bool {
	klines, err := s.klineRepo.FindAllByCoinIdAndIntervalAndCloseTimeLessOrderByOpenTimeWithLimit(coin.Id, viper.GetString("strategy.maResistance.interval"), s.Clock.NowTime(), 1)
	if err != nil {
		zap.S().Errorf("Error during FindClosedAtMoment at %v: %s", s.Clock.NowTime(), err.Error())
		return false
	}

	lastKline := klines[0]

	movingAvgs := s.MovingAverageService.CalculateAvg(coin, viper.GetInt("strategy.maResistance.length.medium"), 1)

	if lastTransaction.FuturesType == constants.LONG {
		if lastKline.Close < movingAvgs[len(movingAvgs)-1] {
			zap.S().Infof("At %v close LONG  below MA price=%v  movingAvgs=%v \n", s.Clock.NowTime(), lastKline.Close, movingAvgs)
			return true
		}
	}

	if lastTransaction.FuturesType == constants.SHORT {
		if lastKline.Close > movingAvgs[len(movingAvgs)-1] {
			zap.S().Infof("At %v close SHORT above MA price=%v  movingAvgs=%v \n", s.Clock.NowTime(), lastKline.Close, movingAvgs)
			return true
		}
	}

	return false
}
