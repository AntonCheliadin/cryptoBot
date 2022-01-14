package util

import (
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
	layout := "2006-01-02"
	t, _ := time.Parse(layout, date)
	return t.UnixNano() / 1000000
}

func ParseDate(date string) (time.Time, error) {
	dateString := strings.Trim(date, " _")
	if dateString == "today" {
		return time.Now(), nil
	}
	if dateString == "yesterday" {
		return time.Now().AddDate(0, 0, -1), nil
	}

	if len(dateString) == 2 {
		return GetDateByDayOfCurrentMonth(dateString)
	}

	layout := "2006-01-02"
	parsedDate, err := time.Parse(layout, dateString)
	return parsedDate, err
}

func GetDateByDayOfCurrentMonth(date string) (time.Time, error) {
	now := time.Now()
	dayInt, err := strconv.Atoi(date)
	return time.Date(now.Year(), now.Month(), dayInt, 0, 0, 0, 0, time.UTC), err
}
