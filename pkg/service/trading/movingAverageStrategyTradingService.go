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

var movingAverageStrategyTradingServiceImpl *MovingAverageStrategyTradingService

func NewMAStrategyTradingService(transactionRepo repository.Transaction, priceChangeRepo repository.PriceChange,
	exchangeApi api.ExchangeApi, clock date.Clock, exchangeDataService *exchange.DataService, klineRepo repository.Kline,
	priceChangeTrackingService *PriceChangeTrackingService, movingAverageService *indicator.MovingAverageService,
	standardDeviationService *indicator.StandardDeviationService) *MovingAverageStrategyTradingService {
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
		MovingAverageService:       movingAverageService,
		StandardDeviationService:   standardDeviationService,
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
	MovingAverageService       *indicator.MovingAverageService
	StandardDeviationService   *indicator.StandardDeviationService
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
	//if openedOrder != nil {
	//	return
	//}
	currentPrice, err := s.ExchangeDataService.GetCurrentPrice(coin)
	if err != nil {
		zap.S().Errorf("Error during GetCurrentCoinPrice at %v: %s", s.Clock.NowTime(), err.Error())
		return
	}

	shortAvgs := s.MovingAverageService.CalculateAvg(coin, viper.GetInt("strategy.ma.length.short"), 2)
	mediumAvgs := s.MovingAverageService.CalculateAvg(coin, viper.GetInt("strategy.ma.length.medium"), 2)

	if shortAvgs == nil || len(shortAvgs) < 2 || mediumAvgs == nil || len(mediumAvgs) < 2 {
		zap.S().Errorf("Can't calculate direction of moving averages")
		return
	}

	if s.isCrossingUp(shortAvgs, mediumAvgs) {
		zap.S().Infof("At %v MA isCrossingUp shortAvgs=[%v] mediumAvgs=[%v]", s.Clock.NowTime(), shortAvgs, mediumAvgs)

		if openedOrder != nil && openedOrder.FuturesType == constants.SHORT {
			zap.S().Infof("Close SHORT and open LONG")
			s.closeOrder(openedOrder, coin)
		}
		if openedOrder == nil || openedOrder.FuturesType == constants.SHORT {
			if currentPrice > shortAvgs[len(shortAvgs)-1] { // if current price above from MA
				//if s.isTrendUp(coin) {
				if s.StandardDeviationService.IsVolatilityOscillatorSignal(coin, viper.GetString("strategy.ma.interval")) {
					s.openOrder(coin, constants.LONG)
				}
			}
		}
		return
	}

	if s.isCrossingDown(shortAvgs, mediumAvgs) {
		zap.S().Infof("At %v MA isCrossingDown shortAvgs=[%v] mediumAvgs=[%v]", s.Clock.NowTime(), shortAvgs, mediumAvgs)

		if openedOrder != nil && openedOrder.FuturesType == constants.LONG {
			zap.S().Infof("Close LONG and open SHORT")
			s.closeOrder(openedOrder, coin)
		}
		if openedOrder == nil || openedOrder.FuturesType == constants.LONG {
			if currentPrice < shortAvgs[len(shortAvgs)-1] { // if current price under from MA
				//if s.isTrendDown(coin) {
				if s.StandardDeviationService.IsVolatilityOscillatorSignal(coin, viper.GetString("strategy.ma.interval")) {
					s.openOrder(coin, constants.SHORT)
				}
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

	//if s.shouldCloseByStopLoss(openedOrder, coin) {
	//	s.closeOrder(openedOrder, coin)
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

func (s *MovingAverageStrategyTradingService) openOrder(coin *domains.Coin, futuresType constants.FuturesType) {
	currentPrice, err := s.ExchangeDataService.GetCurrentPrice(coin)
	if err != nil {
		zap.S().Errorf("Error during GetCurrentCoinPrice at %v: %s", s.Clock.NowTime(), err.Error())
		return
	}
	sumOfProfit, err := s.transactionRepo.CalculateSumOfProfitByCoin(coin.Id, constants.MOVING_AVARAGE)
	amountTransaction := util.CalculateAmountByPriceAndCostWithCents(currentPrice, viper.GetInt64("strategy.ma.cost")+sumOfProfit)
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

func (s *MovingAverageStrategyTradingService) shouldCloseWithProfit(lastTransaction *domains.Transaction, coin *domains.Coin) bool {
	currentPrice, err := s.ExchangeDataService.GetCurrentPrice(coin)
	if err != nil {
		zap.S().Errorf("Error on ExchangeDataService.GetCurrentPrice: %s", err)
		return false
	}

	if lastTransaction.FuturesType == constants.LONG {
		orderProfitInPercent := util.CalculatePercents(lastTransaction.Price, currentPrice)
		if orderProfitInPercent >= viper.GetFloat64("strategy.ma.percentProfit") {
			zap.S().Infof("At %v close LONG with profit price=%v currentProfitInPercent=%v", s.Clock.NowTime(), currentPrice, orderProfitInPercent)
			return true
		}
	}

	if lastTransaction.FuturesType == constants.SHORT {
		orderProfitInPercent := -1 * util.CalculatePercents(lastTransaction.Price, currentPrice)
		if orderProfitInPercent >= viper.GetFloat64("strategy.ma.percentProfit") {
			zap.S().Infof("At %v close SHORT with profit price=%v currentProfitInPercent=%v", s.Clock.NowTime(), currentPrice, orderProfitInPercent)
			return true
		}
	}

	return false
}

func (s *MovingAverageStrategyTradingService) isCloseToBreakeven(lastTransaction *domains.Transaction, coin *domains.Coin) bool {
	currentPrice, err := s.ExchangeDataService.GetCurrentPrice(coin)
	if err != nil {
		zap.S().Errorf("Error on ExchangeDataService.GetCurrentPrice: %s", err)
		return false
	}

	priceChange := s.PriceChangeTrackingService.getChangePrice(lastTransaction.Id, currentPrice)

	if lastTransaction.FuturesType == constants.LONG {
		maxProfitInPercent := util.CalculatePercents(lastTransaction.Price, priceChange.HighPrice)
		currentProfitInPercent := util.CalculatePercents(lastTransaction.Price, currentPrice)

		if maxProfitInPercent > 0.5 && currentProfitInPercent < 0.2 {
			zap.S().Infof("At %v close long  order with price=%v currentProfitInPercent=%v", s.Clock.NowTime(), currentPrice, currentProfitInPercent)
			return true
		}
	}

	if lastTransaction.FuturesType == constants.SHORT {
		maxProfitInPercent := -1 * util.CalculatePercents(lastTransaction.Price, priceChange.LowPrice)
		currentProfitInPercent := -1 * util.CalculatePercents(lastTransaction.Price, currentPrice)

		if maxProfitInPercent > 0.5 && currentProfitInPercent < 0.2 {
			zap.S().Infof("At %v close short order with price=%v currentProfitInPercent=%v", s.Clock.NowTime(), currentPrice, currentProfitInPercent)
			return true
		}
	}

	return false
}

func (s *MovingAverageStrategyTradingService) isCurrentPriceIntersectMA(lastTransaction *domains.Transaction, coin *domains.Coin) bool {
	currentPrice, err := s.ExchangeDataService.GetCurrentPrice(coin)
	if err != nil {
		zap.S().Errorf("Error on ExchangeDataService.GetCurrentPrice: %s", err)
		return false
	}

	movingAvgs := s.MovingAverageService.CalculateAvg(coin, viper.GetInt("strategy.ma.length.medium"), 2)

	if lastTransaction.FuturesType == constants.LONG {
		if currentPrice < movingAvgs[len(movingAvgs)-1] {
			zap.S().Infof("At %v close LONG  below MA price=%v  movingAvgs=%v \n", s.Clock.NowTime(), currentPrice, movingAvgs)
			return true
		}
	}

	if lastTransaction.FuturesType == constants.SHORT {
		if currentPrice > movingAvgs[len(movingAvgs)-1] {
			zap.S().Infof("At %v close SHORT above MA price=%v  movingAvgs=%v \n", s.Clock.NowTime(), currentPrice, movingAvgs)
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

func (s *MovingAverageStrategyTradingService) shouldCloseByStopLoss(lastTransaction *domains.Transaction, coin *domains.Coin) bool {
	currentPrice, err := s.ExchangeDataService.GetCurrentPrice(coin)
	if err != nil {
		zap.S().Errorf("Error on ExchangeDataService.GetCurrentPrice: %s", err)
		return false
	}

	if lastTransaction.FuturesType == constants.LONG {
		orderProfitInPercent := util.CalculatePercents(lastTransaction.Price, currentPrice)
		if orderProfitInPercent <= viper.GetFloat64("strategy.ma.percentStopLoss") {
			zap.S().Infof("at %v close order by stop loss price=%v currentProfitInPercent=%v", s.Clock.NowTime(), currentPrice, orderProfitInPercent)
			return true
		}
	}

	if lastTransaction.FuturesType == constants.SHORT {
		orderProfitInPercent := -1 * util.CalculatePercents(lastTransaction.Price, currentPrice)
		if orderProfitInPercent <= viper.GetFloat64("strategy.ma.percentStopLoss") {
			zap.S().Infof("at %v close order by stop loss price=%v currentProfitInPercent=%v", s.Clock.NowTime(), currentPrice, orderProfitInPercent)
			return true
		}
	}

	return false
}

func (s *MovingAverageStrategyTradingService) isTrendUp(coin *domains.Coin) bool {
	countOfPoints := viper.GetInt("indicator.trend.ma.points")
	longAvgs := s.MovingAverageService.CalculateAvg(coin, viper.GetInt("indicator.trend.ma.length"), countOfPoints)
	if countOfPoints%2 != 0 {
		countOfPoints = countOfPoints + 1
	}

	middleIndex := countOfPoints / 2

	return longAvgs[0] < longAvgs[middleIndex] && longAvgs[middleIndex] < longAvgs[len(longAvgs)-1]
}

func (s *MovingAverageStrategyTradingService) isTrendDown(coin *domains.Coin) bool {
	countOfPoints := viper.GetInt("indicator.trend.ma.points")
	longAvgs := s.MovingAverageService.CalculateAvg(coin, viper.GetInt("indicator.trend.ma.length"), countOfPoints)
	if countOfPoints%2 != 0 {
		countOfPoints = countOfPoints + 1
	}

	middleIndex := countOfPoints / 2

	return longAvgs[0] > longAvgs[middleIndex] && longAvgs[middleIndex] > longAvgs[len(longAvgs)-1]
}

func (s *MovingAverageStrategyTradingService) isProfitByTrolling(lastTransaction *domains.Transaction, coin *domains.Coin) bool {
	currentPrice, err := s.ExchangeDataService.GetCurrentPrice(coin)
	if err != nil {
		zap.S().Errorf("Error on ExchangeDataService.GetCurrentPrice: %s", err)
		return false
	}

	priceChange := s.PriceChangeTrackingService.getChangePrice(lastTransaction.Id, currentPrice)

	if lastTransaction.FuturesType == constants.LONG {
		// close order if price on percentProfit lower from high
		priceChangeInPercent := util.CalculatePercents(priceChange.HighPrice, currentPrice)
		if priceChangeInPercent < -1*viper.GetFloat64("strategy.ma.percentTrollingProfit") {
			zap.S().Infof("At %v close order. Higher price %v current price %v percent %v",
				s.Clock.NowTime(), priceChange.HighPrice, currentPrice, priceChangeInPercent)
			return true
		}
	}
	if lastTransaction.FuturesType == constants.SHORT {
		// close order if price on percentProfit higher from low
		priceChangeInPercent := util.CalculatePercents(priceChange.LowPrice, currentPrice)
		if priceChangeInPercent > viper.GetFloat64("strategy.ma.percentTrollingProfit") {
			zap.S().Infof("At %v close order. Lower price %v current price %v percent %v",
				s.Clock.NowTime(), priceChange.LowPrice, currentPrice, priceChangeInPercent)
			return true
		}
	}

	return false
}
