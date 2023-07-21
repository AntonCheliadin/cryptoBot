package indicator

import (
	"cryptoBot/pkg/constants/futureType"
	"cryptoBot/pkg/data/domains"
	"cryptoBot/pkg/repository"
	"cryptoBot/pkg/service/date"
	"cryptoBot/pkg/service/indicator/techanLib"
	"cryptoBot/pkg/util"
	"github.com/sdcoffey/big"
	"github.com/sdcoffey/techan"
)

var stochasticServiceImpl *StochasticService

func NewStochasticService(clock date.Clock, klineRepo repository.Kline, techanConvertorService *techanLib.TechanConvertorService) *StochasticService {
	if stochasticServiceImpl != nil {
		panic("Unexpected try to create second service instance")
	}
	stochasticServiceImpl = &StochasticService{
		klineRepo:              klineRepo,
		Clock:                  clock,
		TechanConvertorService: techanConvertorService,
	}
	return stochasticServiceImpl
}

type StochasticService struct {
	klineRepo              repository.Kline
	Clock                  date.Clock
	TechanConvertorService *techanLib.TechanConvertorService
}

func (s *StochasticService) CalculateStochasticSignal(coin *domains.Coin, candleDuration string, periodK int, smoothK int, periodD int) (bool, futureType.FuturesType) {
	klinesToFetchSize := util.MinInt(int64(periodK), int64(periodD)) + 10
	series := s.TechanConvertorService.BuildTimeSeriesByKlines(coin, candleDuration, klinesToFetchSize)

	k := techan.NewSimpleMovingAverage(techan.NewFastStochasticIndicator(series, (periodK)), (smoothK))
	d := techan.NewSlowStochasticIndicator(k, (periodD))

	lastK := k.Calculate(int(klinesToFetchSize - 1))
	prevK := k.Calculate(int(klinesToFetchSize - 2))
	prevPrevK := k.Calculate(int(klinesToFetchSize - 3))

	lastD := d.Calculate(int(klinesToFetchSize - 1))
	prevD := d.Calculate(int(klinesToFetchSize - 2))

	isCrossingUp := prevK.LT(prevD) && lastK.GTE(lastD)
	dec20 := big.NewDecimal(20)
	if isCrossingUp && (prevK.LTE(dec20) || prevPrevK.LTE(dec20)) {
		return true, futureType.LONG
	}

	isCrossingDown := prevK.GT(prevD) && lastK.LTE(lastD)
	dec80 := big.NewDecimal(80)
	if isCrossingDown && (prevK.GTE(dec80) || prevPrevK.GTE(dec80)) {
		return true, futureType.SHORT
	}

	if lastK.GTE(big.NewDecimal(50)) {
		return false, futureType.SHORT
	} else {
		return false, futureType.LONG
	}
}
