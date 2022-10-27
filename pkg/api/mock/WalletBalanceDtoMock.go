package mock

type BalanceDtoMock struct{}

func (dto *BalanceDtoMock) GetAvailableBalanceInCents() int64 {
	return 10000
}
