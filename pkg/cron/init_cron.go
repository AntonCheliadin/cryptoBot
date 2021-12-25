package cron

import (
	"github.com/jasonlvhit/gocron"
	"tradingBot/pkg/repository"
	"tradingBot/pkg/service/trading"
)

func InitCronJobs(tradingService trading.TradingService, coinRepository repository.Coin) {
	go func() {
		ch := gocron.Start()

		tradingJob := newTradingJob(tradingService, coinRepository)
		tradingJob.initTradingJob()

		<-ch
	}()
}
