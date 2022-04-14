package configs

import (
	telegramApi "cryptoBot/pkg/api/telegram"
	"time"
)

var RuntimeConfig *config

func NewRuntimeConfig() *config {
	if RuntimeConfig != nil {
		panic("Unexpected try to create second instance")
	}
	RuntimeConfig = &config{
		TradingEnabled: true,
		LimitSpendDay: 500,
	}
	return RuntimeConfig
}

type config struct {
	/**
	Transactions switcher, enable/disable buy and sell transactions.
	*/
	TradingEnabled bool

	/**
	Limit spend money for the last 24 hours.
	 0 - without limit.
	*/
	LimitSpendDay int
}
func (c *config) DisableBuyingForHour() {
	c.TradingEnabled = false
	telegramApi.SendTextToTelegramChat("Trading has been disabled for an hour.")

	select {
	case <-time.After(time.Hour):
		c.TradingEnabled = true
		telegramApi.SendTextToTelegramChat("Trading has been enabled.")
	}
}

func (c *config) HasLimitSpendDay() bool {
	return c.LimitSpendDay > 0
}
