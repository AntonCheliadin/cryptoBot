package main

import (
	"context"
	"cryptoBot"
	"cryptoBot/cmd/bootstrap"
	"cryptoBot/pkg/api/bybit"
	"cryptoBot/pkg/constants"
	"cryptoBot/pkg/controller"
	"cryptoBot/pkg/cron"
	"cryptoBot/pkg/log"
	"cryptoBot/pkg/repository"
	"cryptoBot/pkg/service/date"
	"cryptoBot/pkg/service/exchange"
	"cryptoBot/pkg/service/indicator/techanLib"
	"cryptoBot/pkg/service/orders"
	"cryptoBot/pkg/service/telegram"
	"cryptoBot/pkg/service/trading"
	"fmt"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"os"
	"os/signal"
	"syscall"
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

	zap.S().Info("Trading bot is starting...")

	postgresDb := bootstrap.Database(closableClosure)
	repos := repository.NewRepositories(postgresDb)
	exchangeApi := bybit.NewBybitApi(os.Getenv("BYBIT_CryptoBotFutures_API_KEY"), os.Getenv("BYBIT_CryptoBotFutures_API_SECRET"))
	clock := date.GetClock()

	seriesConvertorService := techanLib.NewTechanConvertorService(clock, repos.Kline)
	exchangeDataService := exchange.NewExchangeDataService(repos.Transaction, repos.Coin, exchangeApi, clock, repos.Kline)
	priceChangeTrackingService := orders.NewPriceChangeTrackingService(repos.PriceChange)
	klinesFetcherService := exchange.NewKlinesFetcherService(exchangeApi, repos.Kline, clock)

	orderManagerService := orders.NewOrderManagerService(repos.Transaction, exchangeApi, clock, exchangeDataService, repos.Kline, constants.PAIR_ARBITRAGE, priceChangeTrackingService,
		orders.NewProfitLossFinderService(clock, repos.Kline),
		0,
		0, 0, 0, 0)

	coins := viper.GetStringSlice("strategy.pairArbitrage.coins")
	for i := 0; i < len(coins); i += 2 {
		symbol1 := coins[i]
		symbol2 := coins[i+1]
		coin1, _ := repos.Coin.FindBySymbol(symbol1)
		coin2, _ := repos.Coin.FindBySymbol(symbol2)

		tradingService := trading.NewPairArbitrageStrategyTradingService(
			repos.Transaction,
			clock,
			exchangeDataService,
			repos.SyntheticKline,
			klinesFetcherService,
			orderManagerService,
			seriesConvertorService,
			coin1,
			coin2,
		)
		cron.InitCronJobs(tradingService)
	}

	var telegramService telegram.ITelegramService //todo implement := telegram.NewTelegramPairTradingService(repos.Transaction, repos.Coin, exchangeApi)

	router := controller.InitControllers(telegramService)

	srv := new(cryptoBot.Server)
	go func() {
		zap.S().Info("Server is doing to be up right now!")
		if err := srv.Run(viper.GetString("server.port"), router); err != nil {
			panic(fmt.Sprintf("Error when starting the http server: %s", err.Error()))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
	<-quit

	zap.S().Info("Logging before Background")
	if err := srv.Shutdown(context.Background()); err != nil {
		zap.S().Errorf("error occured on server shutting down: %s", err.Error())
	}

	if err := postgresDb.Close(); err != nil {
		zap.S().Errorf("error occured on db connection close: %s", err.Error())
	}
}
