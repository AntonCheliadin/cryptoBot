package util

import (
	"math"
)

func CalculateChangeInPercentsAbs(prev, current int64) float64 {
	return math.Abs((float64(current) - float64(prev)) / float64(prev) * 100)
}

func CalculateChangeInPercents(prev, current int64) float64 {
	return (float64(current) - float64(prev)) / float64(prev) * 100
}

func CalculatePercentOf(source float64, percent float64) float64 {
	return source * percent * 0.01
}

func Sum(array []int64) int64 {
	result := int64(0)
	for _, v := range array {
		result += v
	}
	return result
}

// Max returns the larger of x or y.
func Max(x, y int64) int64 {
	if x < y {
		return y
	}
	return x
}

// Min returns the smaller of x or y.
func Min(x, y int64) int64 {
	if x > y {
		return y
	}
	return x
}
