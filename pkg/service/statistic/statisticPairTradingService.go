package statistic

import (
	"cryptoBot/pkg/api"
	"cryptoBot/pkg/constants"
	"cryptoBot/pkg/data/domains"
	"cryptoBot/pkg/repository"
	"cryptoBot/pkg/util"
	"fmt"
	"github.com/spf13/viper"
)

type IStatisticService interface {
	BuildStatistics() string
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

func (s *StatisticPairTradingService) BuildStatistics() string {
	var response = ""

	coins := viper.GetStringSlice("strategy.pairArbitrage.coins")

	var allIds = make([]int64, len(coins)*2)

	for i := 0; i < len(coins); i += 2 {
		symbol1 := coins[i]
		symbol2 := coins[i+1]
		coin1, _ := s.coinRepo.FindBySymbol(symbol1)
		coin2, _ := s.coinRepo.FindBySymbol(symbol2)

		allIds = append(allIds, coin1.Id, coin2.Id)

		response += s.BuildStatisticsByCoins(coin1, coin2)
	}

	return response
}

func (s *StatisticPairTradingService) BuildStatisticsByCoins(coin1 *domains.Coin, coin2 *domains.Coin) string {
	var response = ""

	ids := []int64{coin1.Id, coin2.Id}
	response += "\n" + coin1.Symbol + "-" + coin2.Symbol + "\n"

	rows, err := s.transactionRepo.FetchStatisticByDays(int(constants.PAIR_ARBITRAGE), ids)
	if err != nil {
		return "fetch failed"
	}

	response = "<pre>\n" +
		"|    Date    |   Profit   |   Percent  |    Size    |\n" +
		"|------------|------------|------------|------------|"

	for k := 0; k < len(rows); k += 1 {
		dto := rows[k]
		response += fmt.Sprintf("\n| %v | %10v | %10v | %10v |",
			dto.CreatedAt, util.GetDollarsByCents(dto.ProfitInCents), dto.ProfitPercent, dto.OrdersSize)
	}

	response += "\n</pre>"

	return response
}
