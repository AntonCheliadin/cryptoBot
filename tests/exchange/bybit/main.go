package main

import (
	"cryptoBot/pkg/api"
	"cryptoBot/pkg/api/bybit"
	"cryptoBot/pkg/constants"
	"cryptoBot/pkg/data/domains"
	"cryptoBot/pkg/log"
	"fmt"
	"github.com/joho/godotenv"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"time"
)

func main() {
	if err := godotenv.Load(); err != nil {
		panic(fmt.Sprintf("Failed load env file: %s", err.Error()))
	}
	if err := initConfig(); err != nil {
		panic(fmt.Sprintf("Error during reading configs: %s", err.Error()))
	}

	log.InitLoggerAnalyser()

	exchangeApi := bybit.NewBybitApi().(*bybit.BybitApi)

	coin := &domains.Coin{
		Symbol: "SOLUSDT",
	}

	testGetCurrentPrice(exchangeApi, coin)

	//testGetKlines(exchangeApi, coin)

	//err := exchangeApi.SetFuturesLeverage(coin, 3)
	//if err != nil {
	//	zap.S().Errorf("API error: %s", err.Error())
	//}

	//testOpenFutures(exchangeApi, coin)
	//testCloseFutures(exchangeApi, coin)

	//result, err := exchangeApi.GetActiveOrdersByCoin(coin)
	//if err != nil {
	//	zap.S().Errorf("API error: %s", err.Error())
	//	return
	//}
	//zap.S().Infof("GetActiveOrdersByCoin response: %v", result)

	result, err := exchangeApi.GetWalletBalance()
	if err != nil {
		zap.S().Errorf("API error: %s", err.Error())
		return
	}
	zap.S().Infof("GetWalletBalance response: %v", result)
}

func initConfig() error {
	viper.AddConfigPath("configs")
	viper.SetConfigName("config")
	return viper.ReadInConfig()
}

func testGetCurrentPrice(exchangeApi api.ExchangeApi, coin *domains.Coin) {
	coinPrice, err := exchangeApi.GetCurrentCoinPrice(coin)
	if err != nil {
		zap.S().Errorf("Error on GetCurrentCoinPrice: %s", err)
	}
	fmt.Printf("coinPrice=%s", coinPrice)
}

func testGetKlines(exchangeApi api.ExchangeApi, coin *domains.Coin) {
	timeFrom, _ := time.Parse(constants.DATE_FORMAT, "2022-05-01")
	klinesDto, err := exchangeApi.GetKlines(coin, "1", 10, timeFrom)
	if err != nil {
		zap.S().Errorf("Error on GetCurrentCoinPrice: %s", err)
	}
	fmt.Printf("klinesDto=%v", klinesDto)
}

func testOpenFutures(exchangeApi api.ExchangeApi, coin *domains.Coin) {
	order, err := exchangeApi.OpenFuturesOrder(coin, 2, 3850, constants.LONG)
	if err != nil {
		zap.S().Errorf("API error: %s", err.Error())
		return
	}
	zap.S().Infof("testOpenFutures response: %v", order)
}

func testCloseFutures(exchangeApi api.ExchangeApi, coin *domains.Coin) {
	transaction := domains.Transaction{}
	transaction.Amount = 2
	transaction.FuturesType = constants.LONG
	transaction.Price = 3854

	exchangeApi.CloseFuturesOrder(coin, &transaction, 3836)
}
