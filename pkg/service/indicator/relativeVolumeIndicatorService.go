package indicator

import (
	"cryptoBot/pkg/service/indicator/techanLib"
	"github.com/sdcoffey/big"
	"github.com/sdcoffey/techan"
)

var relativeVolumeIndicatorServiceImpl *RelativeVolumeIndicatorService

func NewRelativeVolumeIndicatorService(techanConvertorService *techanLib.TechanConvertorService) *RelativeVolumeIndicatorService {
	if relativeVolumeIndicatorServiceImpl != nil {
		panic("Unexpected try to create second service instance")
	}
	relativeVolumeIndicatorServiceImpl = &RelativeVolumeIndicatorService{
		TechanConvertorService: techanConvertorService,
		length:                 13,
		thresh:                 11, //, 'Relative Volume Strength Threshold', minval=0)
	}
	return relativeVolumeIndicatorServiceImpl
}

//implemented by for tradingview indicator https://www.tradingview.com/pine/?id=PUB%3BFvJqumctetdFjTc3kvAdgJXrU6fkzHPt
type RelativeVolumeIndicatorService struct {
	TechanConvertorService *techanLib.TechanConvertorService

	length int
	thresh int //= input.int(11, 'Relative Volume Strength Threshold', minval=0)
}

func (s *RelativeVolumeIndicatorService) CalculateRelativeVolumeSignal(series *techan.TimeSeries) bool {
	lastIndex := series.LastIndex()

	/* volAvgL  = ta.sma(nzVolume, length * 5) */
	volumeAvg := techan.NewSimpleMovingAverage(techan.NewVolumeIndicator(series), s.length*5)

	/* volDev   = (volAvgL + 1.618034 * ta.stdev(volAvgL, length * 5)) / volAvgL * thresh / 100 */
	stdDevVolumeAvg := techan.NewStandardDeviationIndicator(volumeAvg)
	volAvgL := volumeAvg.Calculate(lastIndex)
	volDev := (volAvgL.Add(big.NewDecimal(1.618034).Mul(stdDevVolumeAvg.Calculate(lastIndex)))).
		Div(volAvgL.Mul(big.NewFromInt(s.thresh)).
			Div(big.NewDecimal(100)))

	/* volRel   = nzVolume / volAvgL */
	volRel := series.Candles[lastIndex].Volume.Div(volAvgL)

	return volRel.Mul(big.NewDecimal(0.145898)).GT(volDev) //volRel * .145898 > volDev;
}
