package trading

import (
	"cryptoBot/configs"
	"cryptoBot/pkg/api"
	telegramApi "cryptoBot/pkg/api/telegram"
	"cryptoBot/pkg/constants"
	"cryptoBot/pkg/data/domains"
	"cryptoBot/pkg/repository"
	"cryptoBot/pkg/util"
	"database/sql"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"time"
)

type TradingService interface {
	BotAction(coin *domains.Coin)
	InitializeTrading(coin *domains.Coin) error
}

var tradingServiceImpl *HolderStrategyTradingService

func NewHolderStrategyTradingService(transactionRepo repository.Transaction, priceChangeRepo repository.PriceChange, exchangeApi api.ExchangeApi) *HolderStrategyTradingService {
	if tradingServiceImpl != nil {
		panic("Unexpected try to create second service instance")
	}
	tradingServiceImpl = &HolderStrategyTradingService{
		transactionRepo: transactionRepo,
		priceChangeRepo: priceChangeRepo,
		exchangeApi:     exchangeApi,
	}
	return tradingServiceImpl
}

type HolderStrategyTradingService struct {
	transactionRepo repository.Transaction
	priceChangeRepo repository.PriceChange
	exchangeApi     api.ExchangeApi
}

func (s *HolderStrategyTradingService) InitializeTrading(coin *domains.Coin) error {
	return nil
}

func (s *HolderStrategyTradingService) BotAction(coin *domains.Coin) {
	if !configs.RuntimeConfig.TradingEnabled {
		return
	}

	currentPrice, err := s.exchangeApi.GetCurrentCoinPrice(coin)
	if err != nil {
		zap.S().Error(err)
		return
	}

	s.BotActionForPrice(coin, currentPrice)
}

func (s *HolderStrategyTradingService) BotActionForPrice(coin *domains.Coin, currentPrice int64) {
	boughtNotSoldTransaction, err := s.transactionRepo.FindLastBoughtNotSold(coin.Id, constants.HOLDER)
	if err != nil {
		zap.S().Error(err)
		return
	}

	if boughtNotSoldTransaction != nil {
		if s.shouldSell(boughtNotSoldTransaction, currentPrice) {
			s.sell(coin, boughtNotSoldTransaction, currentPrice)
		} else if s.shouldBuy(boughtNotSoldTransaction, currentPrice) {
			s.buy(coin, currentPrice)
		} else {
			zap.S().Debugf("Price change is too small, previous: [%v], current price: [%v], percent: [%v]",
				boughtNotSoldTransaction.Price, currentPrice, s.getPriceChangeInPercent(boughtNotSoldTransaction, currentPrice))
		}
	} else {
		anyLastTransaction, err := s.transactionRepo.FindLastByCoinId(coin.Id, constants.HOLDER)
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

func (s *HolderStrategyTradingService) shouldBuy(lastTransaction *domains.Transaction, currentPrice int64) bool {
	if lastTransaction == nil {
		return true
	}
	tradingPercent := viper.GetFloat64("trading.percentChange")
	priceChangeInPercent := s.getPriceChangeInPercent(lastTransaction, currentPrice)

	if priceChangeInPercent <= tradingPercent*-1 {
		return true
	}

	//priceChange := s.GetChangePrice(lastTransaction.Id, currentPrice)
	//if priceChange.ChangePercents > tradingPercent && util.AlmostEquals(currentPrice, priceChange.LowPrice) {
	//	zap.S().Debugf("High[%v] Low[%v] Percents[%v]. currentPrice[%v]", priceChange.HighPrice, priceChange.LowPrice, priceChange.ChangePercents, currentPrice)
	//	return true
	//}

	return false
}

func (s *HolderStrategyTradingService) shouldSell(lastTransaction *domains.Transaction, currentPrice int64) bool {
	tradingPercent := viper.GetFloat64("trading.percentChange")
	priceChangeInPercent := util.CalculateChangeInPercents(lastTransaction.Price, currentPrice)

	return priceChangeInPercent >= tradingPercent
}

func (s *HolderStrategyTradingService) getPriceChangeInPercent(lastTransaction *domains.Transaction, currentPrice int64) float64 {
	return util.CalculateChangeInPercents(lastTransaction.Price, currentPrice)
}

func (s *HolderStrategyTradingService) buy(coin *domains.Coin, currentPrice int64) {
	if !configs.RuntimeConfig.TradingEnabled {
		return
	}
	if configs.RuntimeConfig.HasLimitSpendDay() {
		var dayAgo = time.Now().AddDate(0, 0, -1)
		spentForTheLast24Hours, err := s.transactionRepo.CalculateSumOfSpentTransactionsAndCreatedAfter(dayAgo, constants.HOLDER)
		if err != nil {
			zap.S().Errorf("Error on CalculateSumOfSpentTransactionsAndCreatedAfter: %s", err)
			return
		}
		if spentForTheLast24Hours > int64(configs.RuntimeConfig.LimitSpendDay)*100 {
			zap.S().Infof("Can't submit buy transactions because of spend limitation. spentForTheLast24Hours = [%s], LimitSpendDay=[%s]", spentForTheLast24Hours, configs.RuntimeConfig.LimitSpendDay)
			return
		}
	}

	amountTransaction := util.CalculateAmountByPriceAndCost(currentPrice, viper.GetInt64("trading.defaultCost"))

	orderDto, err := s.exchangeApi.BuyCoinByMarket(coin, amountTransaction, currentPrice)
	if err != nil || orderDto.GetAmount() == 0 {
		zap.S().Errorf("Error during buy coin by market")
		telegramApi.SendTextToTelegramChat("Error during buy coin by market")
		configs.RuntimeConfig.DisableBuyingForHour()
		return
	}

	s.createBuyTransaction(coin, constants.BUY, orderDto, err)
}

func (s *HolderStrategyTradingService) sell(coin *domains.Coin, buyTransaction *domains.Transaction, currentPrice int64) {
	orderDto, err := s.exchangeApi.SellCoinByMarket(coin, buyTransaction.Amount, currentPrice)
	if err != nil || orderDto.GetAmount() == 0 {
		zap.S().Errorf("Error during sell coin by market")
		telegramApi.SendTextToTelegramChat("Error during sell coin by market")
		return
	}

	sellTransaction := s.createSellTransaction(coin, constants.SELL, orderDto, err, buyTransaction)

	if sellTransaction != nil {
		buyTransaction.RelatedTransactionId = sql.NullInt64{Int64: sellTransaction.Id, Valid: true}
		_ = s.transactionRepo.SaveTransaction(buyTransaction)
	}
}

func (s *HolderStrategyTradingService) createBuyTransaction(coin *domains.Coin, tType constants.TransactionType, orderDto api.OrderResponseDto, apiError error) *domains.Transaction {
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
		zap.S().Errorf("Error during save transaction %s", err)
		return nil
	}

	telegramApi.SendTextToTelegramChat(transaction.String())

	return &transaction
}

func (s *HolderStrategyTradingService) createSellTransaction(coin *domains.Coin, tType constants.TransactionType, orderDto api.OrderResponseDto, apiError error, buyTransaction *domains.Transaction) *domains.Transaction {
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
		zap.S().Errorf("Error during save transaction %s", err.Error())
		return nil
	}

	telegramApi.SendTextToTelegramChat(transaction.String())

	return &transaction
}
