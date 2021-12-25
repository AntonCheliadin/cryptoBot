package util

import "math"

func CalculatePercentsAbs(prev, current int64) float64 {
	return math.Abs((float64(current) - float64(prev)) / float64(prev) * 100)
}

func CalculatePercents(prev, current int64) float64 {
	return (float64(current) - float64(prev)) / float64(prev) * 100
}
