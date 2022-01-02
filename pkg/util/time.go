package util

import (
	"fmt"
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
