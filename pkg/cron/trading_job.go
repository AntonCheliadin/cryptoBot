package cron

import (
	"cryptoBot/pkg/service/trading"
	"github.com/go-co-op/gocron"
	"time"
)

type tradingJob struct {
	tradingService trading.TradingService
}

func newTradingJob(tradingService trading.TradingService) *tradingJob {
	return &tradingJob{tradingService: tradingService}
}

func (j *tradingJob) initTradingJob() {
	s := gocron.NewScheduler(time.UTC)
	s.CronWithSeconds("1 */1 * * * *").Do(j.execute) // every minute
	s.SingletonModeAll()
	s.StartAsync()
}

func (j *tradingJob) execute() {
	j.tradingService.BeforeExecute()
	j.tradingService.Execute()
}
