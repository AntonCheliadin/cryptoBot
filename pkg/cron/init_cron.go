package cron

import (
	"cryptoBot/pkg/service/trading"
	"github.com/jasonlvhit/gocron"
)

func InitCronJobs(tradingService trading.TradingService) {
	go func() {
		ch := gocron.Start()

		tradingJob := newTradingJob(tradingService)
		tradingJob.initTradingJob()

		<-ch
	}()
}
