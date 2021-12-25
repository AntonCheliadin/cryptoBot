package trading

import (
	"database/sql"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"math"
	"tradingBot/pkg/api"
	"tradingBot/pkg/constants"
	"tradingBot/pkg/data/domains"
	"tradingBot/pkg/data/dto/binance"
	"tradingBot/pkg/repository"
	"tradingBot/pkg/util"
)

type TradingService interface {
	BotAction(coin *domains.Coin)
}

var tradingServiceImpl *tradingService

func NewTradingService(transactionRepo repository.Transaction, priceChangeRepo repository.PriceChange, exchangeApi api.ExchangeApi) TradingService {
	if tradingServiceImpl != nil {
		panic("Unexpected try to create second service instance")
	}
	tradingServiceImpl = &tradingService{
		transactionRepo: transactionRepo,
		priceChangeRepo: priceChangeRepo,
		exchangeApi:     exchangeApi,
	}
	return tradingServiceImpl
}

type tradingService struct {
	transactionRepo repository.Transaction
	priceChangeRepo repository.PriceChange
	exchangeApi     api.ExchangeApi
}

func (s *tradingService) BotAction(coin *domains.Coin) {
	currentPrice, err := s.exchangeApi.GetCurrentCoinPrice(coin)
	if err != nil {
		zap.S().Error(err)
		return
	}

	boughtNotSoldTransaction, err := s.transactionRepo.FindLastBoughtNotSold(coin.Id)
	if err != nil {
		zap.S().Error(err)
		return
	}

	if boughtNotSoldTransaction != nil {
		if s.shouldSell(boughtNotSoldTransaction, currentPrice) {
			s.sell(coin, boughtNotSoldTransaction)
		} else if s.shouldBuy(boughtNotSoldTransaction, currentPrice) {
			s.buy(coin, currentPrice)
		} else {
			zap.S().Debugf("Price change is too small, previous: [%v], current price: [%v], percent: [%v]",
				boughtNotSoldTransaction.Price, currentPrice, s.getPriceChangeInPercent(boughtNotSoldTransaction, currentPrice))
		}
	} else {
		anyLastTransaction, err := s.transactionRepo.FindLastByCoinId(coin.Id)
		if err != nil {
			zap.S().Error(err)
			return
		}

		if s.shouldBuy(anyLastTransaction, currentPrice) || s.shouldSell(anyLastTransaction, currentPrice) {
			s.buy(coin, currentPrice)
			return
		} else {
			zap.S().Debugf("Price change is too small, previous: [%v], current price: [%v], percent: [%v]",
				anyLastTransaction.Price, currentPrice, s.getPriceChangeInPercent(anyLastTransaction, currentPrice))
		}
	}
}

func (s *tradingService) shouldBuy(lastTransaction *domains.Transaction, currentPrice int64) bool {
	if lastTransaction == nil {
		return true
	}
	tradingPercent := viper.GetFloat64("trading.percentChange")
	priceChangeInPercent := s.getPriceChangeInPercent(lastTransaction, currentPrice)

	if priceChangeInPercent <= tradingPercent*-1 {
		return true
	}

	priceChange := s.getChangePrice(lastTransaction.Id, currentPrice)
	if priceChange.ChangePercents > tradingPercent && util.AlmostEquals(currentPrice, priceChange.LowPrice) {
		zap.S().Debugf("High[%v] Low[%v] Percents[%v]. currentPrice[%v]", priceChange.HighPrice, priceChange.LowPrice, priceChange.ChangePercents, currentPrice)
		return true
	}

	return false
}

func (s *tradingService) shouldSell(lastTransaction *domains.Transaction, currentPrice int64) bool {
	tradingPercent := viper.GetFloat64("trading.percentChange")
	priceChangeInPercent := s.getPriceChangeInPercent(lastTransaction, currentPrice)

	return priceChangeInPercent >= tradingPercent
}

func (s *tradingService) getPriceChangeInPercent(lastTransaction *domains.Transaction, currentPrice int64) float64 {
	return util.CalculatePercents(lastTransaction.Price, currentPrice)
}

func (s *tradingService) buy(coin *domains.Coin, currentPrice int64) {
	amountTransaction := s.calculateAmountByPriceAndCost(currentPrice, viper.GetInt64("trading.defaultCost"))

	orderDto, err := s.exchangeApi.BuyCoinByMarket(coin, amountTransaction)
	if err != nil || orderDto.GetAmount() == 0 {
		zap.S().Errorf("Error during buy coin by market ", err.Error())
		return
	}

	s.createBuyTransaction(coin, constants.BUY, orderDto, err)
}

func (s *tradingService) calculateAmountByPriceAndCost(currentPrice int64, cost int64) float64 {
	amount := float64(cost) / float64(currentPrice)
	if amount > 10 {
		return math.Round(amount)
	} else if amount > 1 {
		return math.Round(amount*100) / 100
	} else {
		return math.Round(amount*1000000) / 1000000
	}
}

func (s *tradingService) sell(coin *domains.Coin, buyTransaction *domains.Transaction) {
	orderDto, err := s.exchangeApi.SellCoinByMarket(coin, buyTransaction.Amount)
	if err != nil || orderDto.GetAmount() == 0 {
		zap.S().Errorf("Error during sell coin by market ", err.Error())
		return
	}

	sellTransaction := s.createSellTransaction(coin, constants.SELL, orderDto, err, buyTransaction)

	if sellTransaction != nil {
		buyTransaction.RelatedTransactionId = sql.NullInt64{Int64: sellTransaction.Id, Valid: true}
		_ = s.transactionRepo.SaveTransaction(buyTransaction)
	}
}

func (s *tradingService) createBuyTransaction(coin *domains.Coin, tType constants.TransactionType, orderDto *binance.OrderResponseDto, apiError error) *domains.Transaction {
	transaction := domains.Transaction{
		CoinId:          coin.Id,
		TransactionType: tType,
		Amount:          orderDto.GetAmount(),
		Price:           orderDto.CalculateAvgPrice(),
		TotalCost:       orderDto.CalculateTotalCost(),
		Commission:      orderDto.CalculateCommissionInUsd(),
	}

	if apiError != nil {
		transaction.ApiError = sql.NullString{String: apiError.Error(), Valid: true}
	} else {
		transaction.ApiError = sql.NullString{Valid: false}
	}

	if err := s.transactionRepo.SaveTransaction(&transaction); err != nil {
		zap.S().Errorf("Error during save transaction", err.Error())
		return nil
	}

	return &transaction
}

func (s *tradingService) createSellTransaction(coin *domains.Coin, tType constants.TransactionType, orderDto *binance.OrderResponseDto, apiError error, buyTransaction *domains.Transaction) *domains.Transaction {
	sellTotalCost := orderDto.CalculateTotalCost()
	commissionInUsd := orderDto.CalculateCommissionInUsd()

	profitInUsd := sellTotalCost - buyTransaction.TotalCost - commissionInUsd - buyTransaction.Commission

	transaction := domains.Transaction{
		CoinId:               coin.Id,
		TransactionType:      tType,
		Amount:               orderDto.GetAmount(),
		Price:                orderDto.CalculateAvgPrice(),
		TotalCost:            sellTotalCost,
		Commission:           commissionInUsd,
		RelatedTransactionId: sql.NullInt64{Int64: buyTransaction.Id, Valid: true},
		Profit:               sql.NullInt64{Int64: profitInUsd, Valid: true},
		PercentProfit:        sql.NullFloat64{Float64: float64(profitInUsd) / float64(buyTransaction.TotalCost) * 100, Valid: true},
	}

	if apiError != nil {
		transaction.ApiError = sql.NullString{String: apiError.Error(), Valid: true}
	} else {
		transaction.ApiError = sql.NullString{Valid: false}
	}

	if err := s.transactionRepo.SaveTransaction(&transaction); err != nil {
		zap.S().Errorf("Error during save transaction", err.Error())
		return nil
	}

	return &transaction
}

func (s tradingService) getChangePrice(transactionId int64, currentPrice int64) *domains.PriceChange {
	priceChange, _ := s.priceChangeRepo.FindByTransactionId(transactionId)
	if priceChange != nil {
		s.saveNewPriceIfNeeded(priceChange, currentPrice)
	} else {
		priceChange = &domains.PriceChange{
			TransactionId: transactionId,
			LowPrice:      currentPrice,
			HighPrice:     currentPrice,
		}
		priceChange.RecalculatePercent()
		_ = s.priceChangeRepo.SavePriceChange(priceChange)
	}
	return priceChange
}

func (s tradingService) saveNewPriceIfNeeded(priceChange *domains.PriceChange, currentPrice int64) {
	if currentPrice > priceChange.HighPrice {
		priceChange.SetHigh(currentPrice)
		_ = s.priceChangeRepo.SavePriceChange(priceChange)
	} else if currentPrice < priceChange.LowPrice {
		priceChange.SetLow(currentPrice)
		_ = s.priceChangeRepo.SavePriceChange(priceChange)
	}
}
