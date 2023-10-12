package statistic

import (
	"cryptoBot/pkg/api"
	"cryptoBot/pkg/constants"
	"cryptoBot/pkg/data/domains"
	"cryptoBot/pkg/repository"
	"cryptoBot/pkg/util"
	"fmt"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

type IStatisticService interface {
	BuildStatistics() string
	BuildHourStatistics() string
}

var statisticPairTradingServiceImpl *StatisticPairTradingService

func NewStatisticPairTradingService(transactionRepo repository.Transaction, coinRepo repository.Coin, exchangeApi api.ExchangeApi) *StatisticPairTradingService {
	statisticPairTradingServiceImpl = &StatisticPairTradingService{
		transactionRepo: transactionRepo,
		coinRepo:        coinRepo,
		exchangeApi:     exchangeApi,
	}
	return statisticPairTradingServiceImpl
}

type StatisticPairTradingService struct {
	transactionRepo repository.Transaction
	coinRepo        repository.Coin
	exchangeApi     api.ExchangeApi
}

func (s *StatisticPairTradingService) BuildHourStatistics() string {
	coins := viper.GetStringSlice("strategy.pairArbitrage.coins")

	var response = "<pre>\n" +
		"| Coin1 | Coin2 |      Date open      |   Profit   |\n" +
		"|-------|-------|---------------------|------------|"

	for i := 0; i < len(coins); i += 2 {
		symbol1 := coins[i]
		symbol2 := coins[i+1]
		coin1, _ := s.coinRepo.FindBySymbol(symbol1)
		coin2, _ := s.coinRepo.FindBySymbol(symbol2)

		response += s.BuildHourStatisticsByCoins(coin1, coin2)
	}

	response += "\n</pre>"

	return response
}

func (s *StatisticPairTradingService) BuildStatistics() string {
	coins := viper.GetStringSlice("strategy.pairArbitrage.coins")

	var response = "<pre>\n" +
		"| Coin1 | Coin2 |    Date    |   Profit   |   Percent  | Size | O |    Last    |\n" +
		"|-------|-------|------------|------------|------------|------|---|------------|"

	zap.S().Infof("coins %v", coins)

	for i := 0; i < len(coins); i += 2 {
		symbol1 := coins[i]
		symbol2 := coins[i+1]
		coin1, _ := s.coinRepo.FindBySymbol(symbol1)
		coin2, _ := s.coinRepo.FindBySymbol(symbol2)

		response += s.BuildStatisticsByCoins(coin1, coin2)
	}

	response += "\n</pre>"

	return response
}

func (s *StatisticPairTradingService) BuildHourStatisticsByCoins(coin1 *domains.Coin, coin2 *domains.Coin) string {
	openedOrder1, _ := s.transactionRepo.FindOpenedTransactionByCoin((constants.PAIR_ARBITRAGE), coin1.Id)
	openedOrder2, _ := s.transactionRepo.FindOpenedTransactionByCoin((constants.PAIR_ARBITRAGE), coin2.Id)
	if openedOrder1 == nil || openedOrder2 == nil {
		return ""
	}

	currentPrice1, _ := s.exchangeApi.GetCurrentCoinPrice(coin1)
	currentPrice2, _ := s.exchangeApi.GetCurrentCoinPrice(coin2)

	profitInPercent1 := util.CalculateProfitInPercent(openedOrder1.Price, currentPrice1, openedOrder1.FuturesType)
	profitInPercent2 := util.CalculateProfitInPercent(openedOrder2.Price, currentPrice2, openedOrder2.FuturesType)

	sumProfit := profitInPercent1 + profitInPercent2

	return fmt.Sprintf("\n| %5v | %5v | %19v | %9.2f%% |",
		coin1.Symbol[:len(coin1.Symbol)-4],
		coin2.Symbol[:len(coin2.Symbol)-4],
		openedOrder1.CreatedAt.Format(constants.DATE_TIME_FORMAT),
		sumProfit)

}

func (s *StatisticPairTradingService) BuildStatisticsByCoins(coin1 *domains.Coin, coin2 *domains.Coin) string {
	var response = ""

	ids := []int64{coin1.Id, coin2.Id}

	rows, err := s.transactionRepo.FetchStatisticByDays(int(constants.PAIR_ARBITRAGE), ids)
	if err != nil {
		return "\n failed FetchStatisticByDays" + coin1.Symbol + " " + coin2.Symbol
	}

	for k := 0; k < len(rows); k += 1 {
		dto := rows[k]
		response += fmt.Sprintf("\n| %5v | %5v | %10v | %10.2f | %10.2f | %4v |",
			coin1.Symbol[:len(coin1.Symbol)-4],
			coin2.Symbol[:len(coin2.Symbol)-4],
			dto.CreatedAt,
			util.GetDollarsByCents(dto.ProfitInCents),
			dto.ProfitPercent,
			dto.OrdersSize)
	}

	return response
}
