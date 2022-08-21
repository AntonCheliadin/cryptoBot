package mock

import (
	"github.com/spf13/viper"
)

type BalanceDtoMock struct{}

func (dto *BalanceDtoMock) GetAvailableBalanceInCents() int64 {
	return viper.GetInt64("strategy.ma.cost")
}
