package cron

import (
	"github.com/jasonlvhit/gocron"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"tradingBot/pkg/repository"
	"tradingBot/pkg/service/trading"
)

type tradingJob struct {
	tradingService trading.TradingService
	coinRepository repository.Coin
}

func newTradingJob(tradingService trading.TradingService, coinRepository repository.Coin) *tradingJob {
	return &tradingJob{tradingService: tradingService, coinRepository: coinRepository}
}

func (j *tradingJob) initTradingJob() {
	err := gocron.Every(1).Minutes().Do(j.execute)
	if err != nil {
		zap.S().Errorf("Error during trading job %s", err.Error())
	}
}

func (j *tradingJob) execute() {
	coin, err := j.coinRepository.FindBySymbol(viper.GetString("trading.defaultCoin"))

	if err != nil {
		zap.S().Errorf("Error during search coin %s", err.Error())
		return
	}

	j.tradingService.BotAction(coin)
}
