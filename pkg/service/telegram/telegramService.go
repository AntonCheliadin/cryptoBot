package telegram

import (
	"cryptoBot/configs"
	"cryptoBot/pkg/api"
	telegramApi "cryptoBot/pkg/api/telegram"
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
	}
	return "Unexpected command"
}

func (s *TelegramService) buildProfitResponse(command string) string {
	daysString := strings.Trim(command, " ")

	dayInt, err := strconv.Atoi(daysString)
	if dayInt == 0 || err != nil {
		dayInt = 7
	}

	now := time.Now()
	maxDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	dayIterator := maxDate.AddDate(0, 0, -dayInt)

	response := ""

	for dayIterator.Before(maxDate) || dayIterator.Equal(maxDate) {
		profitInCentsByDate, _ := s.transactionRepo.CalculateSumOfProfitByDate(dayIterator)
		spentInCentsByDate, _ := s.transactionRepo.CalculateSumOfSpentTransactionsByDate(dayIterator)

		response += fmt.Sprintf("%v %v %v\n", dayIterator.Format("2006-01-02"),
			util.RoundCentsToUsd(profitInCentsByDate), util.RoundCentsToUsd(spentInCentsByDate))

		dayIterator = dayIterator.AddDate(0, 0, 1)
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
		response += fmt.Sprintf("\n%v spent %v \n", date.Format("2006-01-02"), util.RoundCentsToUsd(spentInCentsByDate))

		profitInCentsByDate, _ := s.transactionRepo.CalculateSumOfProfitByDate(date)
		response += fmt.Sprintf("%v profit %v", date.Format("2006-01-02"), util.RoundCentsToUsd(profitInCentsByDate))

		if spentInCentsByDate > 0 && profitInCentsByDate > 0 {
			response += fmt.Sprintf(" (%.2f %%) \n", (float64(profitInCentsByDate)/float64(spentInCentsByDate))*100)
		}
	}

	return response
}
