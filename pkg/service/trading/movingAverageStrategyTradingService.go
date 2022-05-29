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
	"fmt"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

var movingAverageStrategyTradingServiceImpl *MovingAverageStrategyTradingService

func NewMAStrategyTradingService(transactionRepo repository.Transaction, priceChangeRepo repository.PriceChange,
	exchangeApi api.ExchangeApi, clock date.Clock, exchangeDataService *exchange.DataService, klineRepo repository.Kline,
	priceChangeTrackingService *PriceChangeTrackingService) *MovingAverageStrategyTradingService {
	if movingAverageStrategyTradingServiceImpl != nil {
		panic("Unexpected try to create second service instance")
	}
	movingAverageStrategyTradingServiceImpl = &MovingAverageStrategyTradingService{
		klineRepo:                  klineRepo,
		transactionRepo:            transactionRepo,
		priceChangeRepo:            priceChangeRepo,
		exchangeApi:                exchangeApi,
		Clock:                      clock,
		ExchangeDataService:        exchangeDataService,
		PriceChangeTrackingService: priceChangeTrackingService,
	}
	return movingAverageStrategyTradingServiceImpl
}

type MovingAverageStrategyTradingService struct {
	transactionRepo            repository.Transaction
	priceChangeRepo            repository.PriceChange
	klineRepo                  repository.Kline
	exchangeApi                api.ExchangeApi
	Clock                      date.Clock
	ExchangeDataService        *exchange.DataService
	PriceChangeTrackingService *PriceChangeTrackingService
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
	currentPrice, err := s.ExchangeDataService.GetCurrentPrice(coin)
	if err != nil {
		zap.S().Errorf("Error during GetCurrentCoinPrice: %s", err.Error())
		return
	}

	shortAvgs := s.calculateAvg(coin, viper.GetInt("strategy.ma.length.short"))
	mediumAvgs := s.calculateAvg(coin, viper.GetInt("strategy.ma.length.medium"))

	if shortAvgs == nil || len(shortAvgs) < 2 || mediumAvgs == nil || len(mediumAvgs) < 2 {
		zap.S().Errorf("Can't calculate direction of moving averages")
		return
	}

	if s.isCrossingUp(shortAvgs, mediumAvgs) {
		zap.S().Infof("MA isCrossingUp at %v shortAvgs=[%v] mediumAvgs=[%v]", s.Clock.NowTime(), shortAvgs, mediumAvgs)

		if openedOrder != nil && openedOrder.FuturesType == constants.SHORT {
			zap.S().Infof("Close SHORT and open LONG")
			s.closeOrder(openedOrder, coin)
		}
		if openedOrder == nil || openedOrder.FuturesType == constants.SHORT {
			if currentPrice > shortAvgs[len(shortAvgs)-1] { // if current price above from MA
				s.openOrder(coin, constants.LONG)
			}
		}
		return
	}

	if s.isCrossingDown(shortAvgs, mediumAvgs) {
		zap.S().Infof("MA isCrossingDown at %v shortAvgs=[%v] mediumAvgs=[%v]", s.Clock.NowTime(), shortAvgs, mediumAvgs)

		if openedOrder != nil && openedOrder.FuturesType == constants.LONG {
			zap.S().Infof("Close LONG and open SHORT")
			s.closeOrder(openedOrder, coin)
		}
		if openedOrder == nil || openedOrder.FuturesType == constants.LONG {
			if currentPrice < shortAvgs[len(shortAvgs)-1] { // if current price under from MA
				s.openOrder(coin, constants.SHORT)
			}
		}
		return
	}
}

func (s *MovingAverageStrategyTradingService) closeOrderIfProfitEnough(coin *domains.Coin) {
	openedOrder, _ := s.transactionRepo.FindOpenedTransaction(constants.MOVING_AVARAGE)

	if openedOrder == nil {
		return
	}

	if s.shouldCloseWithProfit(openedOrder, coin) {
		s.closeOrder(openedOrder, coin)
		return
	}
	if s.isCurrentPriceChanged(openedOrder, coin) {
		s.closeOrder(openedOrder, coin)
		return
	}
	//if s.isCurrentPriceIntersectMA(openedOrder, coin) {
	//	s.closeOrder(openedOrder, coin)
	//	return
	//}
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

	zap.S().Infof("at %v Order opened  with price %v and type [%v] (0-L, 1-S)", s.Clock.NowTime(), currentPrice, futuresType)
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

	zap.S().Infof("at %v Order closed with price %v and type [%v] (0-L, 1-S)", s.Clock.NowTime(), currentPrice, closeTransaction.FuturesType)
}

func (s *MovingAverageStrategyTradingService) shouldCloseWithProfit(lastTransaction *domains.Transaction, coin *domains.Coin) bool {
	currentPrice, err := s.ExchangeDataService.GetCurrentPrice(coin)
	if err != nil {
		zap.S().Errorf("Error on ExchangeDataService.GetCurrentPrice: %s", err)
		return false
	}

	if lastTransaction.FuturesType == constants.LONG {
		orderProfitInPercent := util.CalculatePercents(lastTransaction.Price, currentPrice)
		fmt.Printf(" at %v debug long  with profit price=%v currentProfitInPercent=%v \n", s.Clock.NowTime(), currentPrice, orderProfitInPercent)

		if orderProfitInPercent >= viper.GetFloat64("strategy.ma.percentProfit") {
			zap.S().Infof("at %v close order with profit price=%v currentProfitInPercent=%v", s.Clock.NowTime(), currentPrice, orderProfitInPercent)
			return true
		}
	}

	if lastTransaction.FuturesType == constants.SHORT {
		orderProfitInPercent := -1 * util.CalculatePercents(lastTransaction.Price, currentPrice)
		fmt.Printf(" at %v debug short with profit price=%v currentProfitInPercent=%v \n", s.Clock.NowTime(), currentPrice, orderProfitInPercent)

		if orderProfitInPercent >= viper.GetFloat64("strategy.ma.percentProfit") {
			zap.S().Infof("at %v close order with profit price=%v currentProfitInPercent=%v", s.Clock.NowTime(), currentPrice, orderProfitInPercent)
			return true
		}
	}

	return false
}

func (s *MovingAverageStrategyTradingService) isCurrentPriceChanged(lastTransaction *domains.Transaction, coin *domains.Coin) bool {
	currentPrice, err := s.ExchangeDataService.GetCurrentPrice(coin)
	if err != nil {
		zap.S().Errorf("Error on ExchangeDataService.GetCurrentPrice: %s", err)
		return false
	}

	priceChange := s.PriceChangeTrackingService.getChangePrice(lastTransaction.Id, currentPrice)

	if lastTransaction.FuturesType == constants.LONG {
		maxProfitInPercent := util.CalculatePercents(lastTransaction.Price, priceChange.HighPrice)
		currentProfitInPercent := util.CalculatePercents(lastTransaction.Price, currentPrice)

		if maxProfitInPercent > 0.9 && currentProfitInPercent < 0.3 {
			zap.S().Infof("at %v close long  order with price=%v currentProfitInPercent=%v", s.Clock.NowTime(), currentPrice, currentProfitInPercent)
			return true
		}
	}

	if lastTransaction.FuturesType == constants.SHORT {
		maxProfitInPercent := -1 * util.CalculatePercents(lastTransaction.Price, priceChange.LowPrice)
		currentProfitInPercent := -1 * util.CalculatePercents(lastTransaction.Price, currentPrice)

		if maxProfitInPercent > 0.9 && currentProfitInPercent < 0.3 {
			zap.S().Infof("at %v close short order with price=%v currentProfitInPercent=%v", s.Clock.NowTime(), currentPrice, currentProfitInPercent)
			return true
		}
	}

	//if lastTransaction.FuturesType == constants.LONG {
	//	// close order if price on percentProfit lower from high
	//	priceChangeInPercent := util.CalculatePercents(priceChange.HighPrice, currentPrice)
	//	if priceChangeInPercent < -1*viper.GetFloat64("strategy.ma.percentProfit") {
	//		zap.S().Infof("at %v close order. Higher price %v current price %v percent %v profit %v",
	//			s.Clock.NowTime(), priceChange.HighPrice, currentPrice, priceChangeInPercent, orderProfitInPercent)
	//		return true
	//	}
	//}
	//if lastTransaction.FuturesType == constants.SHORT {
	//	// close order if price on percentProfit higher from low
	//	priceChangeInPercent := util.CalculatePercents(priceChange.LowPrice, currentPrice)
	//	if priceChangeInPercent > viper.GetFloat64("strategy.ma.percentProfit") {
	//		zap.S().Infof("at %v close order. Lower price %v current price %v percent %v profit %v",
	//			s.Clock.NowTime(), priceChange.LowPrice, currentPrice, priceChangeInPercent, orderProfitInPercent)
	//		return true
	//	}
	//}

	return false
}

func (s *MovingAverageStrategyTradingService) isCurrentPriceIntersectMA(lastTransaction *domains.Transaction, coin *domains.Coin) bool {
	currentPrice, err := s.ExchangeDataService.GetCurrentPrice(coin)
	if err != nil {
		zap.S().Errorf("Error on ExchangeDataService.GetCurrentPrice: %s", err)
		return false
	}

	movingAvgs := s.calculateAvg(coin, viper.GetInt("strategy.ma.length.long"))

	if lastTransaction.FuturesType == constants.LONG {
		if currentPrice < movingAvgs[len(movingAvgs)-1] {
			fmt.Printf(" at %v debug long  below MA price=%v  movingAvgs=%v \n", s.Clock.NowTime(), currentPrice, movingAvgs)
			return true
		}
	}

	if lastTransaction.FuturesType == constants.SHORT {
		if currentPrice > movingAvgs[len(movingAvgs)-1] {
			fmt.Printf(" at %v debug short above MA price=%v  movingAvgs=%v \n", s.Clock.NowTime(), currentPrice, movingAvgs)
			return true
		}
	}

	return false
}

func (s *MovingAverageStrategyTradingService) isCrossingUp(shortAvgs []int64, mediumAvgs []int64) bool {
	return shortAvgs[0] < mediumAvgs[0] && shortAvgs[1] >= mediumAvgs[1]
}

func (s *MovingAverageStrategyTradingService) isCrossingDown(shortAvgs []int64, mediumAvgs []int64) bool {
	return shortAvgs[0] > mediumAvgs[0] && shortAvgs[1] <= mediumAvgs[1]
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
		CreatedAt:       s.Clock.NowTime(),
	}

	if futuresType == constants.LONG {
		transaction.TransactionType = constants.BUY
	} else {
		transaction.TransactionType = constants.SELL
	}
	return transaction
}

func (s *MovingAverageStrategyTradingService) createCloseTransactionByOrderResponseDto(coin *domains.Coin, openedTransaction *domains.Transaction,
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
		TradingStrategy:      constants.MOVING_AVARAGE,
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
