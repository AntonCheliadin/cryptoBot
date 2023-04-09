package util

import (
	"cryptoBot/pkg/constants/futureType"
	"fmt"
	"go.uber.org/zap"
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

func CalculatePriceForStopLoss(priceInCents int64, stopLossPercent float64, futuresType futureType.FuturesType) int64 {
	percentOfPriceValue := int64(CalculatePercentOf(float64(priceInCents), stopLossPercent))

	result := int64(0)

	if futuresType == futureType.LONG {
		result = priceInCents - percentOfPriceValue
	} else {
		result = priceInCents + percentOfPriceValue
	}

	zap.S().Infof("CalculatePriceForStopLoss price[%v] percent[%v] futuresType[%v] result[%v]", priceInCents, stopLossPercent, futuresType, result)
	return result
}

func CalculatePriceForTakeProfit(priceInCents int64, takeProfitPercent float64, futuresType futureType.FuturesType) int64 {
	percentOfPriceValue := int64(CalculatePercentOf(float64(priceInCents), takeProfitPercent))

	result := int64(0)

	if futuresType == futureType.LONG {
		result = priceInCents + percentOfPriceValue
	} else {
		result = priceInCents - percentOfPriceValue
	}
	zap.S().Infof("CalculatePriceForTakeProfit price[%v] percent[%v] futuresType[%v] result[%v]", priceInCents, takeProfitPercent, futureType.GetString(futuresType), result)
	return result
}

func CalculateProfitInPercent(prevPrice int64, currentPrice int64, futuresType futureType.FuturesType) float64 {
	return CalculateChangeInPercents(prevPrice, currentPrice) * futureType.GetFuturesSignFloat64(futuresType)
}
func CalculateProfitInPercentWithLeverage(prevPrice int64, currentPrice int64, futuresType futureType.FuturesType, leverage int64) float64 {
	return CalculateChangeInPercents(prevPrice, currentPrice) * futureType.GetFuturesSignFloat64(futuresType) * float64(leverage)
}

func CalculateProfitByRation(openPrice int64, stopLossPrice int64, futuresType futureType.FuturesType, profitRatio float64) int64 {
	stopLossInPercent := CalculateChangeInPercentsAbs(openPrice, stopLossPrice)
	takeProfitInPercent := stopLossInPercent * profitRatio

	return CalculatePriceForTakeProfit(openPrice, takeProfitInPercent, futuresType)
}
