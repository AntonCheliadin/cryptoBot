package date

import (
	"time"
)

type Clock interface {
	NowTime() time.Time
}

var clockRealImpl Clock

func GetClock() Clock {
	if clockRealImpl != nil {
		return clockRealImpl
	}

	clockRealImpl = &ClockReal{}

	return clockRealImpl
}

type ClockReal struct {
}

func (c *ClockReal) NowTime() time.Time {
	return time.Now()
}
