package indicator

import (
	"github.com/sdcoffey/techan"
)

var relativeVolumeIndicatorServiceImpl *RelativeVolumeIndicatorService

func NewRelativeVolumeIndicatorService() *RelativeVolumeIndicatorService {
	if relativeVolumeIndicatorServiceImpl != nil {
		panic("Unexpected try to create second service instance")
	}
	relativeVolumeIndicatorServiceImpl = &RelativeVolumeIndicatorService{
		length: 13,
		thresh: 11, //, 'Relative Volume Strength Threshold', minval=0)
	}
	return relativeVolumeIndicatorServiceImpl
}

//implemented by for tradingview indicator https://www.tradingview.com/pine/?id=PUB%3BFvJqumctetdFjTc3kvAdgJXrU6fkzHPt
type RelativeVolumeIndicatorService struct {
	length int
	thresh int //= input.int(11, 'Relative Volume Strength Threshold', minval=0)
}

func (s *RelativeVolumeIndicatorService) CalculateRelativeVolumeSignalWithFloats(series *techan.TimeSeries) bool {
	lastIndex := series.LastIndex()

	/* volAvgL  = ta.sma(nzVolume, length * 5) */
	volumeAvg := techan.NewSimpleMovingAverage(techan.NewVolumeIndicator(series), s.length*5)

	/* volDev   = (volAvgL + 1.618034 * ta.stdev(volAvgL, length * 5)) / volAvgL * thresh / 100 */
	stdDevVolumeAvg := techan.NewWindowedStandardDeviationIndicator(volumeAvg, s.length*5)
	volAvgL := volumeAvg.Calculate(lastIndex).Float()
	volDev := (volAvgL + 1.618034*stdDevVolumeAvg.Calculate(lastIndex).Float()) / volAvgL * float64(s.thresh) / 100

	/* volRel   = nzVolume / volAvgL */
	volRel := series.Candles[lastIndex].Volume.Float() / volAvgL

	return volRel*0.145898 > volDev //volRel * .145898 > volDev;
}
