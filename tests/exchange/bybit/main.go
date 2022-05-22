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

	exchangeApi := bybit.NewBybitApi()

	coin := &domains.Coin{
		Symbol: "SOLUSDT",
	}

	testGetCurrentPrice(exchangeApi, coin)

	testGetKlines(exchangeApi, coin)
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
	timeFrom, _ := time.Parse(constants.DATE_FORMAT, "2022-05-08")
	klinesDto, err := exchangeApi.GetKlines(coin, "15", 10, timeFrom)
	if err != nil {
		zap.S().Errorf("Error on GetCurrentCoinPrice: %s", err)
	}
	fmt.Printf("klinesDto=%s", klinesDto)
}
