package main

import (
	"cryptoBot/cmd/analyser"
	"cryptoBot/cmd/bootstrap"
	"cryptoBot/pkg/api/bybit/mock"
	"cryptoBot/pkg/constants"
	"cryptoBot/pkg/log"
	"cryptoBot/pkg/repository"
	"cryptoBot/pkg/service/date"
	"cryptoBot/pkg/service/exchange"
	"cryptoBot/pkg/service/indicator"
	"cryptoBot/pkg/service/indicator/techanLib"
	"cryptoBot/pkg/service/orders"
	"cryptoBot/pkg/service/trading"
	"github.com/spf13/viper"
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

	mockExchangeApi := mock.NewBybitApiMock()

	clockMock := date.GetClockMock()

	seriesConvertorService := techanLib.NewTechanConvertorService(clockMock, repos.Kline)
	exchangeDataService := exchange.NewExchangeDataService(repos.Transaction, repos.Coin, mockExchangeApi, clockMock, repos.Kline)
	priceChangeTrackingService := orders.NewPriceChangeTrackingService(repos.PriceChange)

	orderManagerService := orders.NewOrderManagerService(repos.Transaction, mockExchangeApi, clockMock, exchangeDataService, repos.Kline, constants.SMA_VOLUME_SCALPER, priceChangeTrackingService,
		orders.NewProfitLossFinderService(clockMock, repos.Kline),
		viper.GetInt64("strategy.smaVolumeScalper.futures.leverage"),
		0, 0, 0, 0)

	klineInterval := 60

	tradingService := trading.NewSmaVolumeScalperStrategyTradingService(
		repos.Transaction,
		clockMock,
		exchangeDataService,
		repos.Kline,
		exchange.NewKlinesFetcherService(mockExchangeApi, repos.Kline, clockMock),
		orderManagerService,
		seriesConvertorService,
		indicator.NewStochasticService(clockMock, repos.Kline, seriesConvertorService),
		indicator.NewSmaTubeService(clockMock, repos.Kline),
		indicator.NewLocalExtremumTrendService(clockMock, repos.Kline),
		indicator.NewRelativeVolumeIndicatorService(),
		klineInterval,
	)
	analyserService := analyser.NewAnalyserRunner(tradingService)

	coin, _ := repos.Coin.FindBySymbol("ETHUSDT")

	analyserService.AnalyseCoin(coin, "2020-11-12", "2023-04-23", klineInterval)
	//BTC from 2020-04-12
	//ETH from 2020-11-12
	//ETC from 2021-07-15
	//ADA from 2021-04-01

	if err := postgresDb.Close(); err != nil {
		zap.S().Errorf("error occured on db connection close: %s", err.Error())
	}

	os.Exit(0)
}
