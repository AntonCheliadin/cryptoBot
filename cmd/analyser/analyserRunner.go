package analyser

import (
	"cryptoBot/pkg/constants"
	"cryptoBot/pkg/data/domains"
	"cryptoBot/pkg/service/date"
	"cryptoBot/pkg/service/trading"
	"time"
)

var analyserRunnerImpl *Runner

func NewAnalyserRunner(tradingService trading.TradingService) *Runner {
	if analyserRunnerImpl != nil {
		panic("Unexpected try to create second service instance")
	}
	analyserRunnerImpl = &Runner{
		tradingService: tradingService,
	}
	return analyserRunnerImpl
}

type Runner struct {
	tradingService trading.TradingService
}

func (runner *Runner) AnalyseCoin(coin *domains.Coin, from string, to string, interval int64) {
	timeMax, _ := time.Parse(constants.DATE_FORMAT, to)
	timeIterator, _ := time.Parse(constants.DATE_FORMAT, from)
	timeIterator = timeIterator.Add(time.Second * 2)

	for ; timeIterator.Before(timeMax); timeIterator = timeIterator.Add(time.Minute * time.Duration(interval)) {
		date.SetMockTime(timeIterator)

		runner.tradingService.BotAction(coin)
	}
}
