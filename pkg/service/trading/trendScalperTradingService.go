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

var trendScalperServiceImpl *TrendScalperTradingService

func NewTrendScalperTradingService(transactionRepo repository.Transaction, priceChangeRepo repository.PriceChange,
	exchangeApi api.ExchangeApi, clock date.Clock, exchangeDataService *exchange.DataService, klineRepo repository.Kline,
	priceChangeTrackingService *PriceChangeTrackingService, movingAverageService *indicator.MovingAverageService,
	standardDeviationService *indicator.StandardDeviationService, klinesFetcherService *exchange.KlinesFetcherService) *TrendScalperTradingService {
	if trendScalperServiceImpl != nil {
		panic("Unexpected try to create second service instance")
	}
	trendScalperServiceImpl = &TrendScalperTradingService{
		klineRepo:                  klineRepo,
		transactionRepo:            transactionRepo,
		priceChangeRepo:            priceChangeRepo,
		exchangeApi:                exchangeApi,
		Clock:                      clock,
		ExchangeDataService:        exchangeDataService,
		PriceChangeTrackingService: priceChangeTrackingService,
		MovingAverageService:       movingAverageService,
		StandardDeviationService:   standardDeviationService,
		KlinesFetcherService:       klinesFetcherService,
	}
	return trendScalperServiceImpl
}

type TrendScalperTradingService struct {
	transactionRepo            repository.Transaction
	priceChangeRepo            repository.PriceChange
	klineRepo                  repository.Kline
	exchangeApi                api.ExchangeApi
	Clock                      date.Clock
	ExchangeDataService        *exchange.DataService
	PriceChangeTrackingService *PriceChangeTrackingService
	MovingAverageService       *indicator.MovingAverageService
	StandardDeviationService   *indicator.StandardDeviationService
	KlinesFetcherService       *exchange.KlinesFetcherService
}

func (s *TrendScalperTradingService) InitializeTrading(coin *domains.Coin) error {
	err := s.exchangeApi.SetFuturesLeverage(coin, viper.GetInt("strategy.trendScalper.futures.leverage"))
	if err != nil {
		return err
	}
	return nil
}

func (s *TrendScalperTradingService) BotAction(coin *domains.Coin) {
	if !configs.RuntimeConfig.TradingEnabled {
		return
	}

	//if s.fetchActualKlines(coin) {
	//	return
	//}

	s.BotSingleAction(coin)
}

func (s *TrendScalperTradingService) BotSingleAction(coin *domains.Coin) {
	s.closeOrderIfProfitEnough(coin)

	s.calculateMovingAverage(coin)
}

func (s *TrendScalperTradingService) calculateMovingAverage(coin *domains.Coin) {
	openedOrder, _ := s.transactionRepo.FindOpenedTransaction(constants.TREND_SCALPER)
	if openedOrder != nil {
		return
	}
	currentPrice, err := s.ExchangeDataService.GetCurrentPrice(coin)
	if err != nil {
		zap.S().Errorf("Error during GetCurrentCoinPrice at %v: %s", s.Clock.NowTime(), err.Error())
		return
	}

	shortAvgs := s.MovingAverageService.CalculateAvg(coin, viper.GetInt("strategy.ma.length.short"), 1)
	mediumAvgs := s.MovingAverageService.CalculateAvg(coin, viper.GetInt("strategy.ma.length.medium"), 1)
	longAvgs := s.MovingAverageService.CalculateAvg(coin, viper.GetInt("strategy.ma.length.long"), 1)

	if shortAvgs == nil || mediumAvgs == nil || longAvgs == nil {
		zap.S().Errorf("Can't calculate direction of moving averages")
		return
	}

	if currentPrice > shortAvgs[len(shortAvgs)-1] { // if current price above from MA
		if s.isTrendUp(coin) {
			s.openOrder(coin, constants.LONG)

		}
	}

	if currentPrice < shortAvgs[len(shortAvgs)-1] { // if current price under from MA
		if s.isTrendDown(coin) {
			s.openOrder(coin, constants.SHORT)
		}
	}
}

func (s *TrendScalperTradingService) isTrendUp(coin *domains.Coin) bool {
	countOfPoints := viper.GetInt("indicator.trend.ma.points")
	longAvgs := s.MovingAverageService.CalculateAvg(coin, viper.GetInt("indicator.trend.ma.length"), countOfPoints)
	if countOfPoints%2 != 0 {
		countOfPoints = countOfPoints + 1
	}

	middleIndex := countOfPoints / 2

	return longAvgs[0] < longAvgs[middleIndex] && longAvgs[middleIndex] < longAvgs[len(longAvgs)-1]
}

func (s *TrendScalperTradingService) isTrendDown(coin *domains.Coin) bool {
	countOfPoints := viper.GetInt("indicator.trend.ma.points")
	longAvgs := s.MovingAverageService.CalculateAvg(coin, viper.GetInt("indicator.trend.ma.length"), countOfPoints)
	if countOfPoints%2 != 0 {
		countOfPoints = countOfPoints + 1
	}

	middleIndex := countOfPoints / 2

	return longAvgs[0] > longAvgs[middleIndex] && longAvgs[middleIndex] > longAvgs[len(longAvgs)-1]
}

func (s *TrendScalperTradingService) closeOrderIfProfitEnough(coin *domains.Coin) {
	openedOrder, _ := s.transactionRepo.FindOpenedTransaction(constants.TREND_SCALPER)

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
}

func (s *TrendScalperTradingService) shouldCloseByStopLoss(lastTransaction *domains.Transaction, coin *domains.Coin) bool {
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

func (s *TrendScalperTradingService) shouldCloseWithProfit(lastTransaction *domains.Transaction, coin *domains.Coin) bool {
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

func (s *TrendScalperTradingService) openOrder(coin *domains.Coin, futuresType constants.FuturesType) {
	currentPrice, err := s.ExchangeDataService.GetCurrentPrice(coin)
	if err != nil {
		zap.S().Errorf("Error during GetCurrentCoinPrice at %v: %s", s.Clock.NowTime(), err.Error())
		return
	}

	amountTransaction := util.CalculateAmountByPriceAndCostWithCents(currentPrice, s.getCostOfOrder())
	orderDto, err2 := s.exchangeApi.OpenFuturesOrder(coin, amountTransaction, currentPrice, futuresType)
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

func (s *TrendScalperTradingService) closeOrder(openTransaction *domains.Transaction, coin *domains.Coin) {
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

func (s *TrendScalperTradingService) getCostOfOrder() int64 {
	walletBalanceDto, err := s.exchangeApi.GetWalletBalance()
	if err != nil {
		zap.S().Errorf("Error during GetWalletBalance at %v: %s", s.Clock.NowTime(), err.Error())
		return 0
	}

	maxOrderCost := walletBalanceDto.GetAvailableBalanceInCents() * viper.GetInt64("strategy.ma.futures.leverage")

	return maxOrderCost
}

func (s *TrendScalperTradingService) createCloseTransactionByOrderResponseDto(coin *domains.Coin, openedTransaction *domains.Transaction,
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
		TradingStrategy:      constants.TREND_SCALPER,
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

func (s *TrendScalperTradingService) createOpenTransactionByOrderResponseDto(coin *domains.Coin, futuresType constants.FuturesType,
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
