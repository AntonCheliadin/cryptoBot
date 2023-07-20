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

func CalculateAmountByPriceAndCost(currentPrice float64, cost float64) float64 {
	amount := float64(cost) / float64(currentPrice)
	if amount > 10 {
		return math.Round(amount)
	} else if amount > 0.1 {
		return math.Round(amount*100) / 100
	} else {
		return math.Round(amount*1000000) / 1000000
	}
}

func CalculatePriceForStopLoss(price float64, stopLossPercent float64, futuresType futureType.FuturesType) float64 {
	percentOfPriceValue := CalculatePercentOf(float64(price), stopLossPercent)

	result := float64(0)

	if futuresType == futureType.LONG {
		result = price - percentOfPriceValue
	} else {
		result = price + percentOfPriceValue
	}

	zap.S().Infof("CalculatePriceForStopLoss price[%v] percent[%v] futuresType[%v] result[%v]", price, stopLossPercent, futuresType, result)
	return result
}

func CalculatePriceForTakeProfit(price float64, takeProfitPercent float64, futuresType futureType.FuturesType) float64 {
	percentOfPriceValue := CalculatePercentOf(float64(price), takeProfitPercent)

	result := float64(0)

	if futuresType == futureType.LONG {
		result = price + percentOfPriceValue
	} else {
		result = price - percentOfPriceValue
	}
	zap.S().Infof("CalculatePriceForTakeProfit price[%v] percent[%v] futuresType[%v] result[%v]", price, takeProfitPercent, futureType.GetString(futuresType), result)
	return result
}

func CalculateProfitInPercent(prevPrice float64, currentPrice float64, futuresType futureType.FuturesType) float64 {
	return CalculateChangeInPercents(prevPrice, currentPrice) * futureType.GetFuturesSignFloat64(futuresType)
}
func CalculateProfitInPercentWithLeverage(prevPrice float64, currentPrice float64, futuresType futureType.FuturesType, leverage int64) float64 {
	return CalculateChangeInPercents(prevPrice, currentPrice) * futureType.GetFuturesSignFloat64(futuresType) * float64(leverage)
}

func CalculateProfitByRation(openPrice float64, stopLossPrice float64, futuresType futureType.FuturesType, profitRatio float64) float64 {
	stopLossInPercent := CalculateChangeInPercentsAbs(openPrice, stopLossPrice)
	takeProfitInPercent := stopLossInPercent * profitRatio

	return CalculatePriceForTakeProfit(openPrice, takeProfitInPercent, futuresType)
}
