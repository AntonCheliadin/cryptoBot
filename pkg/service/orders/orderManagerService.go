package orders

import (
	"cryptoBot/pkg/api"
	telegramApi "cryptoBot/pkg/api/telegram"
	"cryptoBot/pkg/constants"
	"cryptoBot/pkg/constants/futureType"
	"cryptoBot/pkg/data/domains"
	"cryptoBot/pkg/repository"
	"cryptoBot/pkg/service/date"
	"cryptoBot/pkg/service/exchange"
	"cryptoBot/pkg/util"
	"database/sql"
	"fmt"
	"go.uber.org/zap"
	"math"
	"time"
)

var orderManagerServiceImpl *OrderManagerService

func NewOrderManagerService(transactionRepo repository.Transaction, exchangeApi api.ExchangeApi, clock date.Clock,
	exchangeDataService *exchange.DataService, klineRepo repository.Kline, tradingStrategy constants.TradingStrategy,
	priceChangeTrackingService *PriceChangeTrackingService,
	profitLossFinderService *ProfitLossFinderService,
	leverage int64, minProfitForBreakEven float64, closeToEntryForBreakEven float64,
	minTrailingTakeProfitPercent float64, trailingTakeProfitPercent float64) *OrderManagerService {
	if orderManagerServiceImpl != nil {
		panic("Unexpected try to create second service instance")
	}
	orderManagerServiceImpl = &OrderManagerService{
		klineRepo:                    klineRepo,
		transactionRepo:              transactionRepo,
		exchangeApi:                  exchangeApi,
		Clock:                        clock,
		ExchangeDataService:          exchangeDataService,
		tradingStrategy:              tradingStrategy,
		PriceChangeTrackingService:   priceChangeTrackingService,
		leverage:                     leverage,
		minProfitForBreakEven:        minProfitForBreakEven,
		closeToEntryForBreakEven:     closeToEntryForBreakEven,
		minTrailingTakeProfitPercent: minTrailingTakeProfitPercent,
		trailingTakeProfitPercent:    trailingTakeProfitPercent,
		ProfitLossFinderService:      profitLossFinderService,
	}
	return orderManagerServiceImpl
}

type OrderManagerService struct {
	transactionRepo              repository.Transaction
	klineRepo                    repository.Kline
	exchangeApi                  api.ExchangeApi
	Clock                        date.Clock
	ExchangeDataService          *exchange.DataService
	tradingStrategy              constants.TradingStrategy
	PriceChangeTrackingService   *PriceChangeTrackingService
	ProfitLossFinderService      *ProfitLossFinderService
	leverage                     int64
	minProfitForBreakEven        float64
	closeToEntryForBreakEven     float64
	minTrailingTakeProfitPercent float64
	trailingTakeProfitPercent    float64
}

func (s *OrderManagerService) SetFuturesLeverage(coin *domains.Coin, leverage int) error {
	err := s.exchangeApi.SetFuturesLeverage(coin, leverage)
	if err != nil {
		return err
	}
	return nil
}

func (s *OrderManagerService) SetIsolatedMargin(coin *domains.Coin, leverage int) error {
	err := s.exchangeApi.SetIsolatedMargin(coin, leverage)
	if err != nil {
		return err
	}
	return nil
}

func (s *OrderManagerService) OpenFuturesOrderWithPercentStopLoss(coin *domains.Coin, futuresType futureType.FuturesType, stopLossInPercent float64) {
	currentPrice, err := s.ExchangeDataService.GetCurrentPrice(coin)
	if err != nil {
		zap.S().Errorf("Error during GetCurrentCoinPrice at %v: %s", s.Clock.NowTime(), err.Error())
		return
	}

	stopLossPrice := util.CalculatePriceForStopLoss(currentPrice, stopLossInPercent, futuresType)

	s.OpenFuturesOrderWithFixedStopLoss(coin, futuresType, stopLossPrice)
}

func (s *OrderManagerService) OpenFuturesOrderWithCalculateStopLoss(coin *domains.Coin, futuresType futureType.FuturesType, klineLengthInMinutes string) {
	zap.S().Infof("OPEN SIGNAL %v", futureType.GetString(futuresType))

	stopLossPrice, err := s.ProfitLossFinderService.FindStopLoss(coin, s.Clock.NowTime(), klineLengthInMinutes, futuresType)

	if err != nil {
		zap.S().Errorf("Error %s", err.Error())
		return
	}

	s.OpenFuturesOrderWithFixedStopLoss(coin, futuresType, stopLossPrice)
}

func (s *OrderManagerService) OpenFuturesOrderWithFixedStopLoss(coin *domains.Coin, futuresType futureType.FuturesType, stopLossPrice float64) {
	s.openOrderWithCostAndFixedStopLossAndTakeProfit(coin, futuresType, stopLossPrice, 0, s.getCostOfOrder(), constants.FUTURES, false)
}

func (s *OrderManagerService) OpenFuturesOrderWithCostAndFixedStopLossAndTakeProfit(coin *domains.Coin, futuresType futureType.FuturesType, cost float64, stopLossPrice float64, profitPrice float64) {
	s.openOrderWithCostAndFixedStopLossAndTakeProfit(coin, futuresType, stopLossPrice, profitPrice, cost, constants.FUTURES, false)
}

func (s *OrderManagerService) OpenFuturesOrderWithCostAndFixedStopLossAndTakeProfitAndFake(coin *domains.Coin, futuresType futureType.FuturesType, cost float64, stopLossPrice float64, profitPrice float64, isFake bool) {
	s.openOrderWithCostAndFixedStopLossAndTakeProfit(coin, futuresType, stopLossPrice, profitPrice, cost, constants.FUTURES, isFake)
}

func (s *OrderManagerService) OpenFuturesOrderWithCostAndFixedStopLoss(coin *domains.Coin, futuresType futureType.FuturesType, cost float64, stopLossPrice float64) {
	s.openOrderWithCostAndFixedStopLossAndTakeProfit(coin, futuresType, stopLossPrice, 0, cost, constants.FUTURES, false)
}

func (s *OrderManagerService) OpenOrderWithCost(coin *domains.Coin, futuresType futureType.FuturesType, cost float64, tradingType constants.TradingType) {
	s.openOrderWithCostAndFixedStopLossAndTakeProfit(coin, futuresType, 0, 0, cost, tradingType, false)
}

func (s *OrderManagerService) openOrderWithCostAndFixedStopLossAndTakeProfit(coin *domains.Coin, futuresType futureType.FuturesType,
	stopLossPrice float64, takeProfitPrice float64, cost float64, tradingType constants.TradingType, isFake bool) {
	if stopLossPrice > 0 {
		zap.S().Debugf("stopLossPrice %.2f  [%v]", stopLossPrice, s.Clock.NowTime().Format(constants.DATE_TIME_FORMAT))
	}
	if takeProfitPrice > 0 {
		zap.S().Debugf("profitPrice %.2f  [%v]", takeProfitPrice, s.Clock.NowTime().Format(constants.DATE_TIME_FORMAT))
	}

	currentPrice, err := s.ExchangeDataService.GetCurrentPrice(coin)
	if err != nil {
		zap.S().Errorf("Error during GetCurrentCoinPrice at %v: %s", s.Clock.NowTime(), err.Error())
		return
	}

	amountTransaction := util.CalculateAmountByPriceAndCost(currentPrice, cost)
	var orderDto api.OrderResponseDto
	if tradingType == constants.FUTURES {
		orderDto, err = s.exchangeApi.OpenFuturesOrder(coin, amountTransaction, currentPrice, futuresType, stopLossPrice)
	} else if tradingType == constants.SPOT {
		orderDto, err = s.exchangeApi.BuyCoinByMarket(coin, amountTransaction, currentPrice)
	}
	if err != nil {
		zap.S().Errorf("Error during OpenFuturesOrder: %s", err.Error())
		telegramApi.SendTextToTelegramChat(fmt.Sprintf("Error during OpenFuturesOrder: %s", err.Error()))
		return
	}

	transaction := s.createOpenTransactionByOrderResponseDto(coin, futuresType, orderDto, stopLossPrice, takeProfitPrice, isFake)
	if err3 := s.transactionRepo.SaveTransaction(&transaction); err3 != nil {
		zap.S().Errorf("Error during SaveTransaction: %s", err3.Error())
		return
	}

	zap.S().Infof("at %s Order opened [%s] with price %v and type [%v] (0-L, 1-S)", s.Clock.NowTime().Format(constants.DATE_TIME_FORMAT), coin.Symbol, currentPrice, futuresType)
	//telegramApi.SendTextToTelegramChat(coin.Symbol + " " + transaction.String())
}

func (s *OrderManagerService) CloseCombinedOrder(openTransaction []*domains.Transaction, coin *domains.Coin, price float64, tradingType constants.TradingType) {
	for _, transaction := range openTransaction {
		s.CloseOrder(transaction, coin, price, tradingType)
	}
}

func (s *OrderManagerService) CloseFuturesOrderWithCurrentPrice(coin *domains.Coin, openTransaction *domains.Transaction) *domains.Transaction {
	currentPrice, _ := s.ExchangeDataService.GetCurrentPrice(coin)
	return s.CloseOrder(openTransaction, coin, currentPrice, constants.FUTURES)
}

func (s *OrderManagerService) CloseOrder(openTransaction *domains.Transaction, coin *domains.Coin, price float64, tradingType constants.TradingType) *domains.Transaction {
	var orderResponseDto api.OrderResponseDto
	var err error
	if tradingType == constants.SPOT {
		orderResponseDto, err = s.exchangeApi.SellCoinByMarket(coin, openTransaction.Amount, price)
	} else if tradingType == constants.FUTURES {
		orderResponseDto, err = s.exchangeApi.CloseFuturesOrder(coin, openTransaction, price)
	}
	if err != nil {
		zap.S().Errorf("Error during CloseFuturesOrder: %s", err.Error())
		telegramApi.SendTextToTelegramChat(fmt.Sprintf("Error during CloseFuturesOrder: %s", err.Error()))
		return nil
	}

	closeTransaction := s.createCloseTransactionByOrderResponseDto(coin, openTransaction, orderResponseDto)
	if errT := s.transactionRepo.SaveTransaction(closeTransaction); errT != nil {
		zap.S().Errorf("Error during SaveTransaction: %s", errT.Error())
		return nil
	}

	openTransaction.RelatedTransactionId = sql.NullInt64{Int64: closeTransaction.Id, Valid: true}
	_ = s.transactionRepo.SaveTransaction(openTransaction)
	//telegramApi.SendTextToTelegramChat(coin.Symbol + " " + closeTransaction.String())

	return closeTransaction
}

func (s *OrderManagerService) createOpenTransactionByOrderResponseDto(coin *domains.Coin, futuresType futureType.FuturesType,
	orderDto api.OrderResponseDto, stopLossPrice float64, takeProfitPrice float64, isFake bool) domains.Transaction {

	var createdAt time.Time
	if orderDto.GetCreatedAt() != nil {
		createdAt = *orderDto.GetCreatedAt()
	} else {
		createdAt = s.Clock.NowTime().Add(time.Millisecond)
	}

	transaction := domains.Transaction{
		TradingStrategy: s.tradingStrategy,
		FuturesType:     futuresType,
		CoinId:          coin.Id,
		Amount:          orderDto.GetAmount(),
		Price:           orderDto.CalculateAvgPrice(),
		TotalCost:       orderDto.CalculateTotalCost(),
		Commission:      orderDto.CalculateCommissionInUsd(),
		CreatedAt:       createdAt,
		IsFake:          isFake,
	}

	if futuresType == futureType.LONG {
		transaction.TransactionType = constants.BUY
	} else {
		transaction.TransactionType = constants.SELL
	}
	if stopLossPrice > 0 {
		transaction.StopLossPrice = sql.NullFloat64{Float64: stopLossPrice, Valid: true}
	}
	if takeProfitPrice > 0 {
		transaction.TakeProfitPrice = sql.NullFloat64{Float64: takeProfitPrice, Valid: true}
	}
	return transaction
}

func (s *OrderManagerService) createCloseTransactionByOrderResponseDto(coin *domains.Coin, openedTransaction *domains.Transaction,
	orderDto api.OrderResponseDto) *domains.Transaction {

	var buyCost float64
	var sellCost float64
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

	var createdAt time.Time
	if orderDto.GetCreatedAt() != nil {
		createdAt = *orderDto.GetCreatedAt()
	} else {
		createdAt = s.Clock.NowTime()
	}

	percentProfit := float64(profitInUsd) / float64(openedTransaction.TotalCost) * 100

	transaction := domains.Transaction{
		TradingStrategy:      s.tradingStrategy,
		FuturesType:          openedTransaction.FuturesType,
		TransactionType:      transactionType,
		CoinId:               coin.Id,
		Amount:               orderDto.GetAmount(),
		Price:                orderDto.CalculateAvgPrice(),
		TotalCost:            orderDto.CalculateTotalCost(),
		Commission:           orderDto.CalculateCommissionInUsd(),
		RelatedTransactionId: sql.NullInt64{Int64: openedTransaction.Id, Valid: true},
		Profit:               sql.NullInt64{Int64: util.GetCents(profitInUsd), Valid: true},
		PercentProfit:        sql.NullFloat64{Float64: math.Round(percentProfit*100) / 100, Valid: true},
		CreatedAt:            createdAt,
		IsFake:               openedTransaction.IsFake,
	}
	return &transaction
}

func (s *OrderManagerService) getCostOfOrder() float64 {
	walletBalanceDto, err := s.exchangeApi.GetWalletBalance()
	if err != nil {
		zap.S().Errorf("Error during GetWalletBalance at %v: %s", s.Clock.NowTime(), err.Error())
		return 0
	}

	maxOrderCost := walletBalanceDto.GetAvailableBalanceInCents() * float64(s.leverage)

	return maxOrderCost
}

func (s *OrderManagerService) CalculateCurrentProfitInPercentWithoutLeverage(coin *domains.Coin, openedTransaction *domains.Transaction) (float64, error) {
	currentPrice, err := s.ExchangeDataService.GetCurrentPrice(coin)
	if err != nil {
		zap.S().Errorf("Error during GetCurrentCoinPrice at %v: %s", s.Clock.NowTime(), err.Error())
		return 0, err
	}

	currentProfitInPercent := util.CalculateProfitInPercent(openedTransaction.Price, currentPrice, openedTransaction.FuturesType)

	return currentProfitInPercent, nil
}

func (s *OrderManagerService) CalculateCurrentProfitInPercentWithLeverage(coin *domains.Coin, openedTransaction *domains.Transaction) (float64, error) {
	currentPrice, err := s.ExchangeDataService.GetCurrentPrice(coin)
	if err != nil {
		zap.S().Errorf("Error during GetCurrentCoinPrice at %v: %s", s.Clock.NowTime(), err.Error())
		return 0, err
	}

	currentProfitInPercent := util.CalculateProfitInPercentWithLeverage(openedTransaction.Price, currentPrice, openedTransaction.FuturesType, s.leverage)

	return currentProfitInPercent, nil
}

func (s *OrderManagerService) ShouldCloseByTrailingTakeProfitWithoutLeverage(coin *domains.Coin, openedTransaction *domains.Transaction) bool {
	if s.trailingTakeProfitPercent == 0 {
		return false
	}

	currentPrice, err := s.ExchangeDataService.GetCurrentPrice(coin)
	if err != nil {
		zap.S().Errorf("Error during GetCurrentCoinPrice at %v: %s", s.Clock.NowTime(), err.Error())
		return false
	}

	currentProfitInPercent, _ := s.CalculateCurrentProfitInPercentWithoutLeverage(coin, openedTransaction)

	if currentProfitInPercent < s.minTrailingTakeProfitPercent {
		return false
	}

	changePrice := s.PriceChangeTrackingService.GetChangePrice(openedTransaction.Id, currentPrice)
	bestProfitInPercent := s.getBestProfitInPercent(openedTransaction, changePrice)

	return bestProfitInPercent-currentProfitInPercent > s.trailingTakeProfitPercent
}

func (s *OrderManagerService) ShouldCloseByBreakEven(coin *domains.Coin, openedTransaction *domains.Transaction) bool {
	if s.minProfitForBreakEven == 0 {
		return false
	}

	currentPrice, err := s.ExchangeDataService.GetCurrentPrice(coin)
	if err != nil {
		zap.S().Errorf("Error during GetCurrentCoinPrice at %v: %s", s.Clock.NowTime(), err.Error())
		return false
	}

	changePrice := s.PriceChangeTrackingService.GetChangePrice(openedTransaction.Id, currentPrice)
	bestProfitInPercent := s.getBestProfitInPercent(openedTransaction, changePrice)
	currentProfitInPercent := util.CalculateProfitInPercent(openedTransaction.Price, currentPrice, openedTransaction.FuturesType)

	return bestProfitInPercent > s.minProfitForBreakEven && currentProfitInPercent < s.closeToEntryForBreakEven
}

func (s *OrderManagerService) getBestProfitInPercent(openedTransaction *domains.Transaction, changePrice *domains.PriceChange) float64 {
	if openedTransaction.FuturesType == futureType.LONG {
		return util.CalculateProfitInPercent(openedTransaction.Price, util.GetDollarsByCents(changePrice.HighPrice), openedTransaction.FuturesType)
	} else {
		return util.CalculateProfitInPercent(openedTransaction.Price, util.GetDollarsByCents(changePrice.LowPrice), openedTransaction.FuturesType)
	}
}

func (s *OrderManagerService) CreateCloseTransactionOnOrderClosedByExchange(coin *domains.Coin, openedTransaction *domains.Transaction) *domains.Transaction {
	closeTradeRecord, err := s.exchangeApi.GetCloseTradeRecord(coin, openedTransaction)
	if closeTradeRecord == nil || err != nil {
		zap.S().Errorf("Error during GetCloseTradeRecord")
		return nil
	}

	closeTransaction := s.createCloseTransactionByOrderResponseDto(coin, openedTransaction, closeTradeRecord)
	if errT := s.transactionRepo.SaveTransaction(closeTransaction); errT != nil {
		zap.S().Errorf("Error during SaveTransaction: %s", errT.Error())
		return nil
	}

	openedTransaction.RelatedTransactionId = sql.NullInt64{Int64: closeTransaction.Id, Valid: true}
	_ = s.transactionRepo.SaveTransaction(openedTransaction)

	return closeTransaction
}

func (s *OrderManagerService) CloseOpenedOrderByStopLossIfNeeded(coin *domains.Coin, klineInterval string) {
	openedOrder, _ := s.transactionRepo.FindOpenedTransactionByCoin(s.tradingStrategy, coin.Id)
	if openedOrder != nil {
		s.CloseOrderByFixedStopLossOrTakeProfit(coin, openedOrder, klineInterval)
	}
}

func (s *OrderManagerService) CloseOrderByFixedStopLossOrTakeProfit(coin *domains.Coin, openedOrder *domains.Transaction, klineInterval string) *domains.Transaction {
	if isPositionOpened := s.ExchangeDataService.IsPositionOpened(coin, openedOrder); !isPositionOpened && openedOrder != nil {
		return s.CreateCloseTransactionOnOrderClosedByExchange(coin, openedOrder)
	}

	if s.ShouldCloseByStopLoss(openedOrder, klineInterval) {
		return s.CloseOrder(openedOrder, coin, openedOrder.StopLossPrice.Float64, constants.FUTURES)
	}

	if s.ShouldCloseByTakeProfit(openedOrder, klineInterval) {
		return s.CloseOrder(openedOrder, coin, openedOrder.TakeProfitPrice.Float64, constants.FUTURES)
	}
	return nil
}

func (s *OrderManagerService) ShouldCloseByStopLoss(openedTransaction *domains.Transaction, klineInterval string) bool {
	if !openedTransaction.StopLossPrice.Valid || openedTransaction.StopLossPrice.Float64 <= 0 {
		return false
	}

	klines, _ := s.klineRepo.FindAllByCoinIdAndIntervalAndCloseTimeInRange(openedTransaction.CoinId, klineInterval, openedTransaction.CreatedAt, s.Clock.NowTime())

	stopLossInCents := openedTransaction.StopLossPrice.Float64

	for i, kline := range klines {
		if i == 0 {
			continue //the first kline is kline of creation the order
		}
		if openedTransaction.FuturesType == futureType.LONG {
			if kline.Low <= stopLossInCents {
				return true
			}
		} else {
			if kline.High >= stopLossInCents {
				return true
			}
		}
	}

	return false
}

func (s *OrderManagerService) ShouldCloseByTakeProfit(openedTransaction *domains.Transaction, klineInterval string) bool {
	if !openedTransaction.TakeProfitPrice.Valid || openedTransaction.TakeProfitPrice.Float64 <= 0 {
		return false
	}

	klines, _ := s.klineRepo.FindAllByCoinIdAndIntervalAndCloseTimeInRange(openedTransaction.CoinId, klineInterval, openedTransaction.CreatedAt, s.Clock.NowTime())

	takeProfitInCents := openedTransaction.TakeProfitPrice.Float64

	for i, kline := range klines {
		if i == 0 {
			continue //the first kline is kline of creation the order
		}
		if openedTransaction.FuturesType == futureType.LONG {
			if kline.High >= takeProfitInCents {
				return true
			}
		} else {
			if kline.Low <= takeProfitInCents {
				return true
			}
		}
	}

	return false
}
