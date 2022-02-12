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
		buyingEnabled: true,
		LimitSpendDay: 500,
	}
	return RuntimeConfig
}

type config struct {
	/**
	Transactions switcher, enable/disable buy and sell transactions.
	*/
	buyingEnabled bool

	/**
	Limit spend money for the last 24 hours.
	 0 - without limit.
	*/
	LimitSpendDay int
}

func (c *config) IsBuyingEnabled() bool {
	return c.buyingEnabled
}
func (c *config) EnableBuying() {
	c.buyingEnabled = true
}
func (c *config) DisableBuying() {
	c.buyingEnabled = false
}
func (c *config) DisableBuyingForHour() {
	c.buyingEnabled = false
	telegramApi.SendTextToTelegramChat("Trading has been disabled for an hour.")

	select {
	case <-time.After(time.Hour):
		c.EnableBuying()
		telegramApi.SendTextToTelegramChat("Trading has been enabled.")
	}
}

func (c *config) HasLimitSpendDay() bool {
	return c.LimitSpendDay > 0
}
