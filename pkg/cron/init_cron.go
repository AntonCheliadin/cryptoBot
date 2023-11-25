package cron

import (
	"cryptoBot/pkg/service/trading"
)

func InitCronJobs(tradingService trading.TradingService) {
	tradingJob := newTradingJob(tradingService)
	tradingJob.initTradingJob()
}
