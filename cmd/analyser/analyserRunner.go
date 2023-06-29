package analyser

import (
	"cryptoBot/pkg/constants"
	"cryptoBot/pkg/service/date"
	"cryptoBot/pkg/service/trading"
	"time"
)

var analyserRunnerImpl *Runner

func NewAnalyserRunner(tradingService trading.TradingService) *Runner {
	analyserRunnerImpl = &Runner{
		tradingService: tradingService,
	}
	return analyserRunnerImpl
}

type Runner struct {
	tradingService trading.TradingService
}

func (runner *Runner) AnalyseCoin(from string, to string, interval int) {
	timeMax, _ := time.Parse(constants.DATE_FORMAT, to)
	timeIterator, _ := time.Parse(constants.DATE_FORMAT, from)
	timeIterator = timeIterator.Add(time.Second * 2)

	for ; timeIterator.Before(timeMax); timeIterator = timeIterator.Add(time.Minute * time.Duration(interval)) {
		date.SetMockTime(timeIterator)

		runner.tradingService.Execute()
	}
}
