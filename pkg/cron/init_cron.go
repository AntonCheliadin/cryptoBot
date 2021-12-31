package cron

import (
	"cryptoBot/pkg/repository"
	"cryptoBot/pkg/service/trading"
	"github.com/jasonlvhit/gocron"
)

func InitCronJobs(tradingService trading.TradingService, coinRepository repository.Coin) {
	go func() {
		ch := gocron.Start()

		tradingJob := newTradingJob(tradingService, coinRepository)
		tradingJob.initTradingJob()

		<-ch
	}()
}
