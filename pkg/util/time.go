package util

import (
	"cryptoBot/pkg/constants"
	"fmt"
	"strconv"
	"strings"
	"time"
)

func MakeTimestamp() string {
	i := time.Now().UnixNano() / int64(time.Millisecond)
	return fmt.Sprintf("%v", i)
}

func GetMillisByDate(date string) int64 {
	t, _ := time.Parse(constants.DATE_FORMAT, date)
	return GetMillisByTime(t)
}

func GetMillisByTime(date time.Time) int64 {
	return date.UnixNano() / int64(time.Millisecond)
}

func GetSecondsByTime(date time.Time) int {
	return int(date.UnixNano() / int64(time.Second))
}

func GetTimeByMillis(millis int64) time.Time {
	return time.Unix(0, millis*int64(time.Millisecond))
}

func GetTimeBySeconds(seconds int) time.Time {
	return time.Unix(int64(seconds), 0)
}

func ParseDate(date string) (time.Time, error) {
	now := time.Now()
	dateString := strings.Trim(date, " _")

	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	if dateString == "today" {
		return today, nil
	}
	if dateString == "yesterday" {
		return today.AddDate(0, 0, -1), nil
	}

	if len(dateString) == 2 {
		return GetDateByDayOfCurrentMonth(dateString)
	}

	parsedDate, err := time.Parse(constants.DATE_FORMAT, dateString)
	return parsedDate, err
}

func GetDateByDayOfCurrentMonth(date string) (time.Time, error) {
	now := time.Now()
	dayInt, err := strconv.Atoi(date)
	return time.Date(now.Year(), now.Month(), dayInt, 0, 0, 0, 0, time.UTC), err
}

func RoundToMinutes(moment time.Time) time.Time {
	d := (60 * time.Second)
	return moment.Truncate(d)
}

func RoundToMinutesWithInterval(moment time.Time, interval string) time.Time {
	intervalInt, _ := strconv.Atoi(interval)

	d := (60 * time.Second)
	roundTime := moment.Truncate(d)
	if roundTime.Minute()%intervalInt != 0 {
		roundTime = roundTime.Add(time.Minute * time.Duration(moment.Minute()%intervalInt) * -1)
	}

	return roundTime
}

func InTimeSpanInclusive(start, end, check time.Time) bool {
	if start.Before(end) {
		return !check.Before(start) && !check.After(end)
	}
	if start.Equal(end) {
		return check.Equal(start)
	}
	return !start.After(check) || !end.Before(check)
}
func IsTheSameDay(date1, date2 time.Time) bool {
	y1, m1, d1 := date1.Date()
	y2, m2, d2 := date2.Date()
	return y1 == y2 && m1 == m2 && d1 == d2
}
