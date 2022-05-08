package date

import "time"

type ClockMock struct {
	mockTime time.Time
}

var clockMockImpl Clock

func GetClockMock(nowMock time.Time) Clock {
	clockMockImpl = &ClockMock{
		mockTime: nowMock,
	}

	return clockMockImpl
}

func (c *ClockMock) NowTime() time.Time {
	return c.mockTime
}
