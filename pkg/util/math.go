package util

import (
	"math"
)

func CalculateChangeInPercentsAbs(prev, current int64) float64 {
	return math.Abs(CalculateChangeInPercents(prev, current))
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
func SumFloat64(array []float64) float64 {
	result := float64(0)
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

func StandardDeviation(array []float64) float64 {
	var sd float64

	sum := SumFloat64(array)
	avg := sum / float64(len(array))

	for j := 0; j < len(array); j++ {
		sd += math.Pow(array[j]-avg, 2)
	}

	// The use of Sqrt math function func Sqrt(x float64) float64
	return math.Sqrt(sd / float64(len(array)))
}
