package cron

import (
	"cryptoBot/pkg/service/trading"
	"github.com/jasonlvhit/gocron"
	"go.uber.org/zap"
)

type tradingJob struct {
	tradingService trading.TradingService
}

func newTradingJob(tradingService trading.TradingService) *tradingJob {
	return &tradingJob{tradingService: tradingService}
}

func (j *tradingJob) initTradingJob() {
	err := gocron.Every(1).Minutes().Do(j.execute)
	if err != nil {
		zap.S().Errorf("Error during trading job %s", err.Error())
	}
}

func (j *tradingJob) execute() {
	j.tradingService.BeforeExecute()
	j.tradingService.Execute()
}
