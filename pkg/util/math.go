package util

import "math"

func CalculatePercentsAbs(prev, current int64) float64 {
	return math.Abs((float64(current) - float64(prev)) / float64(prev) * 100)
}

func CalculatePercents(prev, current int64) float64 {
	return (float64(current) - float64(prev)) / float64(prev) * 100
}

func Sum(array []int64) int64 {
	result := int64(0)
	for _, v := range array {
		result += v
	}
	return result
}
