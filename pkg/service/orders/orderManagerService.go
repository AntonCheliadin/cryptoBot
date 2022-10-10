package orders

import (
	"cryptoBot/pkg/api"
	"cryptoBot/pkg/constants"
	"cryptoBot/pkg/constants/futureType"
	"cryptoBot/pkg/data/domains"
	"cryptoBot/pkg/repository"
	"cryptoBot/pkg/service/date"
	"cryptoBot/pkg/service/exchange"
	"cryptoBot/pkg/util"
	"database/sql"
	"go.uber.org/zap"
)

var orderManagerServiceImpl *OrderManagerService

func NewOrderManagerService(transactionRepo repository.Transaction, exchangeApi api.ExchangeApi, clock date.Clock,
	exchangeDataService *exchange.DataService, klineRepo repository.Kline, tradingStrategy constants.TradingStrategy,
	priceChangeTrackingService *PriceChangeTrackingService,
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

func (s *OrderManagerService) OpenOrderWithPercentStopLoss(coin *domains.Coin, futuresType futureType.FuturesType, stopLossInPercent float64) {
	currentPrice, err := s.ExchangeDataService.GetCurrentPrice(coin)
	if err != nil {
		zap.S().Errorf("Error during GetCurrentCoinPrice at %v: %s", s.Clock.NowTime(), err.Error())
		return
	}

	stopLossPrice := util.CalculatePriceForStopLoss(currentPrice, stopLossInPercent, futuresType)

	s.OpenOrderWithFixedStopLoss(coin, futuresType, stopLossPrice)
}

func (s *OrderManagerService) OpenOrderWithFixedStopLoss(coin *domains.Coin, futuresType futureType.FuturesType, stopLossPriceInCents int64) {
	currentPrice, err := s.ExchangeDataService.GetCurrentPrice(coin)
	if err != nil {
		zap.S().Errorf("Error during GetCurrentCoinPrice at %v: %s", s.Clock.NowTime(), err.Error())
		return
	}

	amountTransaction := util.CalculateAmountByPriceAndCostWithCents(currentPrice, s.getCostOfOrder())
	orderDto, err2 := s.exchangeApi.OpenFuturesOrder(coin, amountTransaction, currentPrice, futuresType, stopLossPriceInCents)
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

func (s *OrderManagerService) CloseOrder(openTransaction *domains.Transaction, coin *domains.Coin, price int64) {
	orderResponseDto, err := s.exchangeApi.CloseFuturesOrder(coin, openTransaction, price)
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

func (s *OrderManagerService) createOpenTransactionByOrderResponseDto(coin *domains.Coin, futuresType futureType.FuturesType,
	orderDto api.OrderResponseDto) domains.Transaction {
	transaction := domains.Transaction{
		TradingStrategy: s.tradingStrategy,
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

func (s *OrderManagerService) createCloseTransactionByOrderResponseDto(coin *domains.Coin, openedTransaction *domains.Transaction,
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
		TradingStrategy:      s.tradingStrategy,
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

func (s *OrderManagerService) getCostOfOrder() int64 {
	walletBalanceDto, err := s.exchangeApi.GetWalletBalance()
	if err != nil {
		zap.S().Errorf("Error during GetWalletBalance at %v: %s", s.Clock.NowTime(), err.Error())
		return 0
	}

	maxOrderCost := walletBalanceDto.GetAvailableBalanceInCents() * s.leverage

	return maxOrderCost
}

func (s *OrderManagerService) ShouldCloseByTrailingTakeProfit(coin *domains.Coin, openedTransaction *domains.Transaction) bool {
	if s.trailingTakeProfitPercent == 0 {
		return false
	}

	currentPrice, err := s.ExchangeDataService.GetCurrentPrice(coin)
	if err != nil {
		zap.S().Errorf("Error during GetCurrentCoinPrice at %v: %s", s.Clock.NowTime(), err.Error())
		return false
	}

	currentProfitInPercent := util.CalculateProfitInPercent(openedTransaction.Price, currentPrice, openedTransaction.FuturesType)

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
		return util.CalculateProfitInPercent(openedTransaction.Price, changePrice.HighPrice, openedTransaction.FuturesType)
	} else {
		return util.CalculateProfitInPercent(openedTransaction.Price, changePrice.LowPrice, openedTransaction.FuturesType)
	}
}
