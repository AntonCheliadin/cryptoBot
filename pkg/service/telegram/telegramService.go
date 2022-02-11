package telegram

import (
	"cryptoBot/configs"
	"cryptoBot/pkg/api"
	telegramApi "cryptoBot/pkg/api/telegram"
	"cryptoBot/pkg/constants"
	"cryptoBot/pkg/data/dto/telegram"
	"cryptoBot/pkg/repository"
	"cryptoBot/pkg/util"
	"fmt"
	"github.com/spf13/viper"
	"strconv"
	"strings"
	"time"
)

const COMMAND_STATS string = "/stats"
const COMMAND_PROFIT string = "/profit"
const COMMAND_BUY_STOP string = "/stop_buying"
const COMMAND_BUY_START string = "/start_buying"
const COMMAND_LIMIT_SPEND string = "/limit_spend"

var telegramServiceImpl *TelegramService

func NewTelegramService(transactionRepo repository.Transaction, coinRepo repository.Coin, exchangeApi api.ExchangeApi) *TelegramService {
	if telegramServiceImpl != nil {
		panic("Unexpected try to create second service instance")
	}
	telegramServiceImpl = &TelegramService{
		transactionRepo: transactionRepo,
		coinRepo:        coinRepo,
		exchangeApi:     exchangeApi,
	}
	return telegramServiceImpl
}

type TelegramService struct {
	transactionRepo repository.Transaction
	coinRepo        repository.Coin
	exchangeApi     api.ExchangeApi
}

func (s *TelegramService) HandleMessage(update *telegram.Update) {
	var response = s.buildResponse(update)

	telegramApi.SendTextToTelegramChat(response)
}

func (s *TelegramService) buildResponse(update *telegram.Update) string {
	if strings.HasPrefix(update.Message.Text, COMMAND_STATS) {
		return s.buildStatistics(strings.ReplaceAll(update.Message.Text, COMMAND_STATS, ""))
	} else if strings.HasPrefix(update.Message.Text, COMMAND_PROFIT) {
		return s.buildProfitResponse(strings.ReplaceAll(update.Message.Text, COMMAND_PROFIT, ""))
	} else if COMMAND_BUY_STOP == update.Message.Text {
		configs.RuntimeConfig.DisableBuying()
		return "BuyingEnabled = " + strconv.FormatBool(configs.RuntimeConfig.IsBuyingEnabled())
	} else if COMMAND_BUY_START == update.Message.Text {
		configs.RuntimeConfig.EnableBuying()
		return "BuyingEnabled = " + strconv.FormatBool(configs.RuntimeConfig.IsBuyingEnabled())
	} else if strings.HasPrefix(update.Message.Text, COMMAND_LIMIT_SPEND) {
		isNewLimitSet := s.setLimit(update.Message.Text)
		if isNewLimitSet {
			return "New limit is set"
		} else {
			return "New limit is not set"
		}
	}
	return "Unexpected command"
}

func (s TelegramService) setLimit(limitInputValue string) bool {
	limitString := strings.Trim(limitInputValue, " ")

	limitInt, err := strconv.Atoi(limitString)
	if limitInt < 0 || err != nil {
		return false
	}

	configs.RuntimeConfig.LimitSpendDay = limitInt
	return true
}

func (s *TelegramService) buildProfitResponse(command string) string {
	daysString := strings.Trim(command, " ")

	shortResult := false

	if strings.Contains(daysString, "short") {
		shortResult = true
		daysString = strings.ReplaceAll(daysString, "short", "")
	}

	dayInt, err := strconv.Atoi(daysString)
	if dayInt == 0 || err != nil {
		dayInt = 7
	}

	now := time.Now()
	maxDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	dayIterator := maxDate.AddDate(0, 0, -dayInt)

	response := "<pre>\n" +
		"|    Date    |  Profit  |\n" +
		"|------------|----------|"

	if !shortResult {
		response = "<pre>\n" +
			"|    Date    |  Profit  |   Bought   |    Sold    |  Not sold  |  Min price |\n" +
			"|------------|----------|------------|------------|------------|------------|"
	}

	sumProfit := int64(0)
	sumNotSold := int64(0)
	sumBought := int64(0)
	sumSold := int64(0)

	for dayIterator.Before(maxDate) || dayIterator.Equal(maxDate) {
		profitInCentsByDate, _ := s.transactionRepo.CalculateSumOfProfitByDate(dayIterator)
		sumProfit += profitInCentsByDate

		response += fmt.Sprintf("\n| %v | %8v |", dayIterator.Format(constants.DATE_FORMAT),
			util.RoundCentsToUsd(profitInCentsByDate))

		if !shortResult {
			spentInCentsByDate, _ := s.transactionRepo.CalculateSumOfSpentTransactionsByDate(dayIterator)
			boughtInCentsByDate, _ := s.transactionRepo.CalculateSumOfTransactionsByDateAndType(dayIterator, constants.BUY)
			soldInCentsByDate, _ := s.transactionRepo.CalculateSumOfTransactionsByDateAndType(dayIterator, constants.SELL)
			minPrice, _ := s.transactionRepo.FindMinPriceByDate(dayIterator)
			sumNotSold += spentInCentsByDate
			sumBought += boughtInCentsByDate
			sumSold += soldInCentsByDate

			response += fmt.Sprintf(" %10v | %10v | %10v | %10v |",
				util.RoundCentsToUsd(boughtInCentsByDate), util.RoundCentsToUsd(soldInCentsByDate),
				util.RoundCentsToUsd(spentInCentsByDate), util.RoundCentsToUsd(minPrice))
		}

		dayIterator = dayIterator.AddDate(0, 0, 1)
	}

	if !shortResult {
		response += fmt.Sprintf("\n|------------|----------|------------|------------|------------|------------|\n|   total    | %8v | %10v | %10v | %10v | %10v |\n</pre>",
			util.RoundCentsToUsd(sumProfit), util.RoundCentsToUsd(sumBought), util.RoundCentsToUsd(sumSold), util.RoundCentsToUsd(sumNotSold), "")
	} else {
		response += fmt.Sprintf("\n|------------|----------|\n|   total    | %8v |\n</pre>",
			util.RoundCentsToUsd(sumProfit))
	}
	return response
}

func (s *TelegramService) buildStatistics(command string) string {
	var response = ""

	coin, _ := s.coinRepo.FindBySymbol(viper.GetString("trading.defaultCoin"))
	currentPrice, _ := s.exchangeApi.GetCurrentCoinPrice(coin)
	boughtNotSoldTransaction, _ := s.transactionRepo.FindLastBoughtNotSold(coin.Id)

	if boughtNotSoldTransaction != nil && currentPrice > 0 {
		response += fmt.Sprintf("last bought for %v \ncurrent price %v (%.2f%%) \n",
			util.RoundCentsToUsd(boughtNotSoldTransaction.Price), util.RoundCentsToUsd(currentPrice), util.CalculatePercents(boughtNotSoldTransaction.Price, currentPrice))
	}

	spentInCents, _ := s.transactionRepo.CalculateSumOfSpentTransactions()
	response += "\ntotal spent " + util.RoundCentsToUsd(spentInCents) + "\n"

	profitInCents, _ := s.transactionRepo.CalculateSumOfProfit()
	response += "total profit " + util.RoundCentsToUsd(profitInCents)

	if profitInCents > 0 && spentInCents > 0 {
		response += fmt.Sprintf(" (%.2f%%) \n", (float64(profitInCents)/float64(spentInCents))*100)
	}

	if date, err := util.ParseDate(command); err == nil {
		spentInCentsByDate, _ := s.transactionRepo.CalculateSumOfSpentTransactionsByDate(date)
		response += fmt.Sprintf("\n%v spent %v \n", date.Format(constants.DATE_FORMAT), util.RoundCentsToUsd(spentInCentsByDate))

		profitInCentsByDate, _ := s.transactionRepo.CalculateSumOfProfitByDate(date)
		response += fmt.Sprintf("%v profit %v", date.Format(constants.DATE_FORMAT), util.RoundCentsToUsd(profitInCentsByDate))

		if spentInCentsByDate > 0 && profitInCentsByDate > 0 {
			response += fmt.Sprintf(" (%.2f %%) \n", (float64(profitInCentsByDate)/float64(spentInCentsByDate))*100)
		}
	}

	return response
}
