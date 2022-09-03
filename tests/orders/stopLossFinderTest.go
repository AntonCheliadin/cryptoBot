package main

import (
	"cryptoBot/pkg/constants"
	"cryptoBot/pkg/service/orders"
	"fmt"
	"github.com/spf13/viper"
)

func main() {
	if err := initConfig(); err != nil {
		panic(fmt.Sprintf("Error during reading configs: %s", err.Error()))
	}

	serviceMock := orders.NewProfitLossFinderService(nil, nil)

	data := []int64{
		10000, 10000, 10000, 9950, 10050,
		10000, 9800, 10200, 9776, 10225,
		10000, 9600, 10400, 9700, 10300,
	}

	for i := 0; i < len(data); i += 5 {
		currentPrice := data[i]
		minLow := data[i+1]
		maxHigh := data[i+2]
		expectedResultLong := data[i+3]
		expectedResultShort := data[i+4]

		stopLossPriceLong := serviceMock.GetStopLossInConfigRange(currentPrice, minLow, maxHigh, constants.LONG)
		stopLossPriceShort := serviceMock.GetStopLossInConfigRange(currentPrice, minLow, maxHigh, constants.SHORT)
		fmt.Printf("%v -- expected: %v; actual: %v \n", expectedResultLong == stopLossPriceLong, expectedResultLong, stopLossPriceLong)
		fmt.Printf("%v -- expected: %v; actual: %v \n\n", expectedResultShort == stopLossPriceShort, expectedResultShort, stopLossPriceShort)
	}

}

func initConfig() error {
	viper.AddConfigPath("configs")
	viper.SetConfigName("config")
	return viper.ReadInConfig()
}
