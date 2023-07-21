package mock

type BalanceDtoMock struct{}

func (dto *BalanceDtoMock) GetAvailableBalanceInCents() float64 {
	return 100.00
}
