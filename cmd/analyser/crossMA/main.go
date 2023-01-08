package main

import (
	"cryptoBot/cmd/bootstrap"
	"cryptoBot/pkg/api/bybit/mock"
	"cryptoBot/pkg/log"
	"cryptoBot/pkg/repository"
	"cryptoBot/pkg/service/analyser"
	"cryptoBot/pkg/service/date"
	"cryptoBot/pkg/service/exchange"
	"cryptoBot/pkg/service/indicator"
	"cryptoBot/pkg/service/indicator/techanLib"
	"cryptoBot/pkg/service/orders"
	"cryptoBot/pkg/service/trading"
	"go.uber.org/zap"
	"os"
)

func main() {
	bootstrap.Run()
	log.InitLoggerAnalyser()

	var closableClosure []func()

	defer func() {
		for i := range closableClosure {
			closableClosure[i]()
		}
	}()

	postgresDb := bootstrap.Database(closableClosure)

	repos := repository.NewRepositories(postgresDb)

	//exchangeApi := binance.NewBinanceApi()
	//mockExchangeApi := mock.NewBinanceApiMock()

	//exchangeApi := bybit.NewBybitApi()
	mockExchangeApi := mock.NewBybitApiMock()

	//tradingService := trading.NewHolderStrategyTradingService(repos.Transaction, repos.PriceChange, mockExchangeApi)
	//analyserService := analyser.NewAnalyserService(repos.Transaction, repos.PriceChange, exchangeApi, tradingService)

	maService := indicator.NewMovingAverageService(date.GetClock(), repos.Kline)
	seriesConvertorService := techanLib.NewTechanConvertorService(date.GetClock(), repos.Kline)
	stdDevService := indicator.NewStandardDeviationService(date.GetClock(), repos.Kline, seriesConvertorService)
	exchangeDataService := exchange.NewExchangeDataService(repos.Transaction, repos.Coin, mockExchangeApi, date.GetClock(), repos.Kline)
	priceChangeTrackingService := orders.NewPriceChangeTrackingService(repos.PriceChange)
	fetcherService := exchange.NewKlinesFetcherService(mockExchangeApi, repos.Kline, date.GetClock())

	maTradingService := trading.NewMAStrategyTradingService(repos.Transaction, repos.PriceChange, mockExchangeApi, date.GetClock(), exchangeDataService, repos.Kline, priceChangeTrackingService, maService, stdDevService, fetcherService)
	analyserService := analyser.NewMovingAverageStrategyAnalyserService(repos.Transaction, repos.PriceChange, mockExchangeApi, maTradingService, repos.Kline)

	//maResistanceTradingService := trading.NewMovingAverageResistanceStrategyTradingService(repos.Transaction, repos.PriceChange, mockExchangeApi, date.GetClock(), exchangeDataService, repos.Kline, priceChangeTrackingService, maService)
	//analyserService := analyser.NewMovingAverageResistanceStratagyAnalyserService(repos.Transaction, repos.PriceChange, mockExchangeApi, maResistanceTradingService, repos.Kline)

	coin, _ := repos.Coin.FindBySymbol("SOLUSDT")

	analyserService.AnalyseCoin(coin, "2022-01-10", "2022-08-27") //max interval  2022-03-04 2022-07-28

	if err := postgresDb.Close(); err != nil {
		zap.S().Errorf("error occured on db connection close: %s", err.Error())
	}

	os.Exit(0)
}
