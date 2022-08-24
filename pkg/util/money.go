package util

import (
	"cryptoBot/pkg/constants"
	"fmt"
	"math"
	"strconv"
)

func GetCentsFromString(money string) int64 {
	parseFloat, _ := strconv.ParseFloat(money, 64)
	return int64(parseFloat * 100)
}

func GetCents(money float64) int64 {
	return int64(money * 100)
}

func RoundCentsToUsd(moneyInCents int64) string {
	return fmt.Sprintf("$%.2f", float64(moneyInCents)/100)
}

func GetDollarsByCents(moneyInCents int64) float64 {
	return float64(moneyInCents) / 100
}

/* AlmostEquals(50_000, 50_010) == true */
func AlmostEquals(money1 int64, money2 int64) bool {
	changedInPercents := CalculateChangeInPercentsAbs(money2, money1)

	return changedInPercents < 0.02
}

func CalculateAmountByPriceAndCost(currentPriceWithCents int64, costWithoutCents int64) float64 {
	amount := float64(costWithoutCents*100) / float64(currentPriceWithCents)
	if amount > 10 {
		return math.Round(amount)
	} else if amount > 0.1 {
		return math.Round(amount*100) / 100
	} else {
		return math.Round(amount*1000000) / 1000000
	}
}

func CalculateAmountByPriceAndCostWithCents(currentPriceWithCents int64, costWithCents int64) float64 {
	amount := float64(costWithCents) / float64(currentPriceWithCents)
	if amount > 10 {
		return math.Round(amount)
	} else if amount > 0.1 {
		return math.Round(amount*100) / 100
	} else {
		return math.Round(amount*1000000) / 1000000
	}
}

func CalculatePriceForStopLoss(priceInCents int64, stopLossPercent float64, futuresType constants.FuturesType) int64 {
	percentOfPriceValue := int64(CalculatePercentOf(float64(priceInCents), stopLossPercent))

	if futuresType == constants.LONG {
		return priceInCents + percentOfPriceValue
	} else {
		return priceInCents - percentOfPriceValue
	}
}
