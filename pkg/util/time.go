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
	return t.UnixNano() / 1000000
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
