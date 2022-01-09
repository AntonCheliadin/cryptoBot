package telegram

import (
	telegramApi "cryptoBot/pkg/api/telegram"
	"cryptoBot/pkg/data/dto/telegram"
	"cryptoBot/pkg/repository"
	"cryptoBot/pkg/util"
	"fmt"
	"strings"
)

const STATS_COMMAND string = "/stats"

var telegramServiceImpl *TelegramService

func NewTelegramService(transactionRepo repository.Transaction) *TelegramService {
	if telegramServiceImpl != nil {
		panic("Unexpected try to create second service instance")
	}
	telegramServiceImpl = &TelegramService{
		transactionRepo: transactionRepo,
	}
	return telegramServiceImpl
}

type TelegramService struct {
	transactionRepo repository.Transaction
}

func (s *TelegramService) HandleMessage(update *telegram.Update) {
	var response = s.buildResponse(update)

	telegramApi.SendTextToTelegramChat(response)
}

func (s *TelegramService) buildResponse(update *telegram.Update) string {
	if strings.HasPrefix(update.Message.Text, STATS_COMMAND) {
		return s.buildStatistics(strings.ReplaceAll(update.Message.Text, STATS_COMMAND, ""))
	}
	return "Unexpected command"
}

func (s *TelegramService) buildStatistics(command string) string {
	var response = "stats:\n"

	if spentInCents, err := s.transactionRepo.CalculateSumOfSpentTransactions(); err == nil {
		response += "total spent " + util.RoundCentsToUsd(spentInCents) + "\n"
	}

	if profitInCents, err := s.transactionRepo.CalculateSumOfProfit(); err == nil {
		response += "total profit " + util.RoundCentsToUsd(profitInCents) + "\n"
	}

	if date, err := util.ParseDate(command); err == nil {
		if spentInCentsByDate, err := s.transactionRepo.CalculateSumOfSpentTransactionsByDate(date); err == nil {
			response += fmt.Sprintf("%v spent %v \n", date.Format("2006-01-02"), util.RoundCentsToUsd(spentInCentsByDate))
		}

		if profitInCentsByDate, err := s.transactionRepo.CalculateSumOfProfitByDate(date); err == nil {
			response += fmt.Sprintf("%v profit %v \n", date.Format("2006-01-02"), util.RoundCentsToUsd(profitInCentsByDate))
		}
	}

	return response
}
