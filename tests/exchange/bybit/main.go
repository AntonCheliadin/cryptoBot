package main

import (
	"cryptoBot/pkg/api"
	"cryptoBot/pkg/api/bybit"
	"cryptoBot/pkg/constants/futureType"
	"cryptoBot/pkg/data/domains"
	"cryptoBot/pkg/log"
	"cryptoBot/pkg/util"
	"database/sql"
	"fmt"
	"github.com/joho/godotenv"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"os"
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

	exchangeApi := bybit.NewBybitApi(os.Getenv("BYBIT_PairTrading1_API_KEY"), os.Getenv("BYBIT_PairTrading1_API_SECRET")).(*bybit.BybitApi)

	coin := &domains.Coin{
		Symbol: "DASHUSDT",
	}

	testGetCurrentPrice(exchangeApi, coin)
	//testGetCurrentPriceForFutures(exchangeApi, coin)

	//testGetKlines(exchangeApi, coin)
	//testGetKlinesFutures(exchangeApi, coin)

	//err := exchangeApi.SetFuturesLeverage(coin, 1)
	//if err != nil {
	//	zap.S().Errorf("API error: %s", err.Error())
	//}

	//err := exchangeApi.SetIsolatedMargin(coin, 1)
	//if err != nil {
	//	zap.S().Errorf("API error: %s", err.Error())
	//}

	//testOpenFutures(exchangeApi, coin)
	//testGetActiveFuturesOrder(exchangeApi, coin, "722d4949-395d-45c9-b128-d7afc823870e")
	//testBreakEvenOrder(exchangeApi, coin)
	//testReplaceOrder(exchangeApi, coin)
	//testCloseFutures(exchangeApi, coin)

	//testGetPosition(exchangeApi, coin)
	//testGetCloseTradeRecord(exchangeApi, coin)
	//testGetTradesRecord(exchangeApi, coin)
	//testGetConditionalOrder(exchangeApi, coin)

	//testBuySpot(exchangeApi, coin)
	//testSellSpot(exchangeApi, coin)

	//result, err := exchangeApi.GetFuturesActiveOrdersByCoin(coin)
	//if err != nil {
	//	zap.S().Errorf("API error: %s", err.Error())
	//	return
	//}
	//zap.S().Infof("GetFuturesActiveOrdersByCoin response: %v", result)

	//result, err := exchangeApi.GetWalletBalance()
	//if err != nil {
	//	zap.S().Errorf("API error: %s", err.Error())
	//	return
	//}
	//zap.S().Infof("GetWalletBalance response: %v", result)
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
	fmt.Printf("coinPrice=%v\n", coinPrice)
}

func testGetCurrentPriceForFutures(exchangeApi api.ExchangeApi, coin *domains.Coin) {
	coinPrice, err := exchangeApi.GetCurrentCoinPriceForFutures(coin)
	if err != nil {
		zap.S().Errorf("Error on GetCurrentPriceForFutures: %s", err)
	}
	fmt.Printf("coinPrice futures=%v\n", coinPrice)
}

func testGetKlines(exchangeApi api.ExchangeApi, coin *domains.Coin) {
	timeFrom := time.Now().Add(time.Minute * time.Duration(-60))
	klinesDto, err := exchangeApi.GetKlines(coin, "1", 200, timeFrom)
	if err != nil {
		zap.S().Errorf("Error on GetCurrentCoinPrice: %s", err)
	}
	fmt.Printf("klinesDto=%v\n", klinesDto)
	fmt.Printf("klinesDto current price spot =%v\n", klinesDto.GetKlines()[0].GetClose())
	return
}

func testGetKlinesFutures(exchangeApi api.ExchangeApi, coin *domains.Coin) api.KlinesDto {
	timeFrom := time.Now().Add(time.Minute * time.Duration(-60))
	klinesDto, err := exchangeApi.GetKlinesFutures(coin, "1", 200, timeFrom)
	if err != nil {
		zap.S().Errorf("Error on GetKlinesFutures: %s", err)
	}
	fmt.Printf("klinesDto=%v\n", klinesDto)
	fmt.Printf("klinesDto current price futures=%v\n", klinesDto.GetKlines()[0].GetClose())
	return klinesDto
}

func testOpenFutures(exchangeApi api.ExchangeApi, coin *domains.Coin) api.OrderResponseDto {
	order, err := exchangeApi.OpenFuturesOrder(coin, 40, 29, futureType.LONG, 26)
	if err != nil {
		zap.S().Errorf("API error: %s", err.Error())
		return nil
	}
	zap.S().Infof("testOpenFutures response: %v", order)
	return order
}

func testCloseFutures(exchangeApi api.ExchangeApi, coin *domains.Coin) {
	transaction := domains.Transaction{}
	transaction.Amount = 40
	transaction.FuturesType = futureType.LONG
	transaction.Price = 31

	exchangeApi.CloseFuturesOrder(coin, &transaction, 3836)
}

func testGetActiveFuturesOrder(exchangeApi api.ExchangeApi, coin *domains.Coin, orderId string) {
	activeFuturesOrder, err := exchangeApi.GetLastFuturesOrder(coin, orderId)
	if err != nil {
		zap.S().Errorf("API error: %s", err.Error())
		return
	}
	zap.S().Infof("testGetActiveFuturesOrder response: %v", activeFuturesOrder)
}

func testBreakEvenOrder(exchangeApi *bybit.BybitApi, coin *domains.Coin) {
	responseDto, err := exchangeApi.OpenFuturesConditionalOrder(coin, 2, 3123, 3123, 3140, futureType.SHORT)
	if err != nil {
		zap.S().Errorf("API error: %s", err.Error())
		return
	}
	zap.S().Infof("testBreakEvenOrder response: %v", responseDto)
}

func testReplaceOrder(exchangeApi *bybit.BybitApi, coin *domains.Coin) {
	transaction := domains.Transaction{}
	transaction.ClientOrderId = sql.NullString{String: "56cb70c8-563e-4916-8c3b-31b68f7303a9"}

	responseDto, err := exchangeApi.ReplaceFuturesActiveOrder(coin, &transaction, 3131)
	if err != nil {
		zap.S().Errorf("API error: %s", err.Error())
		return
	}
	zap.S().Infof("testBreakEvenOrder response: %v", responseDto)
}

func testGetPosition(exchangeApi *bybit.BybitApi, coin *domains.Coin) {
	responseDto, err := exchangeApi.GetPosition(coin)
	if err != nil {
		zap.S().Errorf("API error: %s", err.Error())
		return
	}
	zap.S().Infof("response: %v", responseDto)
}
func testGetCloseTradeRecord(exchangeApi *bybit.BybitApi, coin *domains.Coin) {
	transaction := domains.Transaction{}
	transaction.CreatedAt = util.GetTimeByMillis(1665567144610)
	transaction.FuturesType = futureType.LONG
	transaction.Amount = 2

	responseDto, err := exchangeApi.GetCloseTradeRecord(coin, &transaction)
	if err != nil {
		zap.S().Errorf("API error: %s", err.Error())
		return
	}
	zap.S().Infof("response amount=%v price=%v fee=%v cost=%v",
		responseDto.GetAmount(), responseDto.CalculateAvgPrice(), responseDto.CalculateCommissionInUsd(), responseDto.CalculateTotalCost())
}

func testGetTradesRecord(exchangeApi *bybit.BybitApi, coin *domains.Coin) {
	transaction := domains.Transaction{}
	transaction.CreatedAt = util.GetTimeByMillis(1665567144610) //time.Date(2022, 10, 12, 12, 32, 24, 0, time.UTC)
	transaction.FuturesType = futureType.LONG
	transaction.Amount = 2

	responseDto, err := exchangeApi.GetTradeRecords(coin, &transaction)
	if err != nil {
		zap.S().Errorf("API error: %s", err.Error())
		return
	}
	zap.S().Infof("response: %v", responseDto)
}

func testGetConditionalOrder(exchangeApi *bybit.BybitApi, coin *domains.Coin) {
	responseDto, err := exchangeApi.GetConditionalOrder(coin)
	if err != nil {
		zap.S().Errorf("API error: %s", err.Error())
		return
	}
	zap.S().Infof("response: %v", responseDto)
}

func testBuySpot(exchangeApi api.ExchangeApi, coin *domains.Coin) api.OrderResponseDto {
	order, err := exchangeApi.BuyCoinByMarket(coin, 0.03, 1645*100)
	if err != nil {
		zap.S().Errorf("API error: %s", err.Error())
		return nil
	}
	zap.S().Infof("testBuySpot response: %s", order)
	return order
}

func testSellSpot(exchangeApi api.ExchangeApi, coin *domains.Coin) api.OrderResponseDto {
	order, err := exchangeApi.SellCoinByMarket(coin, 0.03021, 1645*100)
	if err != nil {
		zap.S().Errorf("API error: %s", err.Error())
		return nil
	}
	zap.S().Infof("testSellSpot response: %s", order)
	return order
}
