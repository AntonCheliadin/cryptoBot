package trading

import (
	"cryptoBot/configs"
	"cryptoBot/pkg/api"
	"cryptoBot/pkg/constants"
	"cryptoBot/pkg/data/domains"
	"cryptoBot/pkg/repository"
	"cryptoBot/pkg/service/date"
	"cryptoBot/pkg/service/exchange"
	"cryptoBot/pkg/util"
	"database/sql"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

var movingAverageStrategyTradingServiceImpl *MovingAverageStrategyTradingService

func NewMAStrategyTradingService(transactionRepo repository.Transaction, priceChangeRepo repository.PriceChange,
	exchangeApi api.ExchangeApi, clock date.Clock, exchangeDataService *exchange.DataService, klineRepo repository.Kline) *MovingAverageStrategyTradingService {
	if movingAverageStrategyTradingServiceImpl != nil {
		panic("Unexpected try to create second service instance")
	}
	movingAverageStrategyTradingServiceImpl = &MovingAverageStrategyTradingService{
		klineRepo:           klineRepo,
		transactionRepo:     transactionRepo,
		priceChangeRepo:     priceChangeRepo,
		exchangeApi:         exchangeApi,
		Clock:               clock,
		ExchangeDataService: exchangeDataService,
	}
	return movingAverageStrategyTradingServiceImpl
}

type MovingAverageStrategyTradingService struct {
	transactionRepo     repository.Transaction
	priceChangeRepo     repository.PriceChange
	klineRepo           repository.Kline
	exchangeApi         api.ExchangeApi
	Clock               date.Clock
	ExchangeDataService *exchange.DataService
}

func (s *MovingAverageStrategyTradingService) BotAction(coin *domains.Coin) {
	if !configs.RuntimeConfig.TradingEnabled {
		return
	}

	//todo fetch needed bars from bybit

	s.BotSingleAction(coin)
}

func (s *MovingAverageStrategyTradingService) BotSingleAction(coin *domains.Coin) {
	s.closeOrderIfProfitEnough(coin)

	if s.Clock.NowTime().Minute()%viper.GetInt("strategy.ma.interval") == 0 {
		s.calculateMovingAverage(coin)
	}
}

func (s *MovingAverageStrategyTradingService) calculateMovingAverage(coin *domains.Coin) {
	openedOrder, _ := s.transactionRepo.FindOpenedTransaction(constants.MOVING_AVARAGE)

	shortAvgs := s.calculateAvg(coin, viper.GetInt("strategy.ma.length.short"))
	mediumAvgs := s.calculateAvg(coin, viper.GetInt("strategy.ma.length.medium"))

	if shortAvgs == nil || len(shortAvgs) < 2 || mediumAvgs == nil || len(mediumAvgs) < 2 {
		zap.S().Errorf("Can't calculate direction of moving averages")
		return
	}

	if s.isCrossingUp(shortAvgs, mediumAvgs) {
		if openedOrder != nil && openedOrder.FuturesType == constants.SHORT {
			s.closeOrder(openedOrder, coin)
		}
		s.openOrder(coin, constants.LONG)
		return
	}

	if s.isCrossingDown(shortAvgs, mediumAvgs) {
		if openedOrder != nil && openedOrder.FuturesType == constants.LONG {
			s.closeOrder(openedOrder, coin)
		}
		s.openOrder(coin, constants.SHORT)
		return
	}
}

func (s *MovingAverageStrategyTradingService) closeOrderIfProfitEnough(coin *domains.Coin) {
	openedOrder, _ := s.transactionRepo.FindOpenedTransaction(constants.MOVING_AVARAGE)

	if openedOrder != nil {
		if s.shouldCloseWithProfit(openedOrder, coin) {
			s.closeOrder(openedOrder, coin)
		}
	}
}

func (s *MovingAverageStrategyTradingService) openOrder(coin *domains.Coin, futuresType constants.FuturesType) {
	currentPrice, err := s.ExchangeDataService.GetCurrentPrice(coin)
	if err != nil {
		zap.S().Errorf("Error during GetCurrentCoinPrice: %s", err.Error())
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
}

func (s *MovingAverageStrategyTradingService) closeOrder(openTransaction *domains.Transaction, coin *domains.Coin) {
	currentPrice, err := s.ExchangeDataService.GetCurrentPrice(coin)
	if err != nil {
		zap.S().Errorf("Error during GetCurrentCoinPrice: %s", err.Error())
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

func (s *MovingAverageStrategyTradingService) shouldCloseWithProfit(lastTransaction *domains.Transaction, coin *domains.Coin) bool {
	currentPrice, err := s.ExchangeDataService.GetCurrentPrice(coin)
	if err != nil {
		zap.S().Errorf("Error on ExchangeDataService.GetCurrentPrice: %s", err)
		return false
	}

	priceChangeInPercent := util.CalculatePercents(lastTransaction.Price, currentPrice)

	if lastTransaction.FuturesType == constants.SHORT {
		priceChangeInPercent = priceChangeInPercent * (-1)
	}

	return priceChangeInPercent >= viper.GetFloat64("strategy.ma.percentProfit")
}

func (s *MovingAverageStrategyTradingService) isCrossingUp(shortAvgs []int64, mediumAvgs []int64) bool {
	return shortAvgs[0] < mediumAvgs[0] && shortAvgs[1] > mediumAvgs[1]
}

func (s *MovingAverageStrategyTradingService) isCrossingDown(shortAvgs []int64, mediumAvgs []int64) bool {
	return shortAvgs[0] > mediumAvgs[0] && shortAvgs[1] < mediumAvgs[1]
}

/**
return two last points of moving averages
*/
func (s *MovingAverageStrategyTradingService) calculateAvg(coin *domains.Coin, length int) []int64 {
	candleDuration := viper.GetString("strategy.ma.interval")
	klines, err := s.klineRepo.FindAllByCoinIdAndIntervalAndCloseTimeLessOrderByOpenTimeWithLimit(coin.Id, candleDuration, s.Clock.NowTime(), int64(length+1))
	if err != nil {
		zap.S().Errorf("Error on FindAllByCoinIdAndIntervalAndCloseTimeLessOrderByOpenTimeWithLimit: %s", err)
		return nil
	}

	var avgPoints []int64
	var movingAvgPoints []int64

	for _, kline := range klines {
		avgPoints = append(avgPoints, (kline.Open+kline.Close+kline.High+kline.Low)/4)

		if len(avgPoints) == length {
			averageByLength := util.Sum(avgPoints) / int64(length)
			movingAvgPoints = append(movingAvgPoints, averageByLength)

			avgPoints = avgPoints[1:] //remove first element
		}
	}

	return movingAvgPoints
}

func (s *MovingAverageStrategyTradingService) createOpenTransactionByOrderResponseDto(coin *domains.Coin, futuresType constants.FuturesType,
	orderDto api.OrderResponseDto) domains.Transaction {
	transaction := domains.Transaction{
		TradingStrategy: constants.MOVING_AVARAGE,
		FuturesType:     futuresType,
		CoinId:          coin.Id,
		Amount:          orderDto.GetAmount(),
		Price:           orderDto.CalculateAvgPrice(),
		TotalCost:       orderDto.CalculateTotalCost(),
		Commission:      orderDto.CalculateCommissionInUsd(),
	}

	if futuresType == constants.LONG {
		transaction.TransactionType = constants.BUY
	} else {
		transaction.TransactionType = constants.SELL
	}
	return transaction
}

func (s *MovingAverageStrategyTradingService) createCloseTransactionByOrderResponseDto(coin *domains.Coin, lastTransaction *domains.Transaction,
	orderDto api.OrderResponseDto) domains.Transaction {

	profitInUsd := orderDto.CalculateTotalCost() - lastTransaction.TotalCost - orderDto.CalculateCommissionInUsd() - lastTransaction.Commission

	transaction := domains.Transaction{
		TradingStrategy:      constants.MOVING_AVARAGE,
		FuturesType:          lastTransaction.FuturesType,
		CoinId:               coin.Id,
		Amount:               orderDto.GetAmount(),
		Price:                orderDto.CalculateAvgPrice(),
		TotalCost:            orderDto.CalculateTotalCost(),
		Commission:           orderDto.CalculateCommissionInUsd(),
		RelatedTransactionId: sql.NullInt64{Int64: lastTransaction.Id, Valid: true},
		Profit:               sql.NullInt64{Int64: profitInUsd, Valid: true},
		PercentProfit:        sql.NullFloat64{Float64: float64(profitInUsd) / float64(lastTransaction.TotalCost) * 100, Valid: true},
	}

	if lastTransaction.FuturesType == constants.LONG {
		transaction.TransactionType = constants.SELL
	} else {
		transaction.TransactionType = constants.BUY
	}
	return transaction
}
