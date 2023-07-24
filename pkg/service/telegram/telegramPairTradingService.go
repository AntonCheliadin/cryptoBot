package telegram

import (
	"cryptoBot/pkg/api"
	telegramApi "cryptoBot/pkg/api/telegram"
	"cryptoBot/pkg/data/dto/telegram"
	"cryptoBot/pkg/repository"
	"cryptoBot/pkg/service/statistic"
	"strings"
)

var telegramPairTradingServiceImpl ITelegramService

func NewTelegramPairTradingService(transactionRepo repository.Transaction, coinRepo repository.Coin,
	exchangeApi api.ExchangeApi, statisticPairTradingService statistic.IStatisticService) ITelegramService {
	if telegramPairTradingServiceImpl != nil {
		panic("Unexpected try to create second service instance")
	}
	telegramPairTradingServiceImpl = &TelegramPairTradingService{
		transactionRepo:  transactionRepo,
		coinRepo:         coinRepo,
		exchangeApi:      exchangeApi,
		statisticService: statisticPairTradingService,
	}
	return telegramPairTradingServiceImpl
}

type TelegramPairTradingService struct {
	transactionRepo  repository.Transaction
	coinRepo         repository.Coin
	exchangeApi      api.ExchangeApi
	statisticService statistic.IStatisticService
}

func (s *TelegramPairTradingService) HandleMessage(update *telegram.Update) {
	var response = s.buildResponse(update)

	telegramApi.SendTextToTelegramChat(response)
}

func (s *TelegramPairTradingService) buildResponse(update *telegram.Update) string {
	if strings.HasPrefix(update.Message.Text, COMMAND_STATS) {
		return s.statisticService.BuildStatistics()
	}
	return "Unexpected command"
}
