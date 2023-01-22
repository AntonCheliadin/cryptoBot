package date

import "time"

type ClockMock struct {
	mockTime time.Time
}

var clockMockImpl Clock

func NewClockMock(nowMock time.Time) Clock {
	clockMockImpl = &ClockMock{
		mockTime: nowMock,
	}

	return clockMockImpl
}

func (c *ClockMock) NowTime() time.Time {
	return c.mockTime
}

func (c *ClockMock) SetTime(nowMockTime time.Time) {
	c.mockTime = nowMockTime
}

func GetClockMock() Clock {
	if clockMockImpl != nil {
		return clockMockImpl
	}

	clockMockImpl = &ClockMock{}

	return clockMockImpl
}

func SetMockTime(nowMockTime time.Time) {
	clockMockImpl.SetTime(nowMockTime)
}
