package main

import (
	"context"
	"cryptoBot"
	"cryptoBot/configs"
	"cryptoBot/pkg/api/bybit"
	"cryptoBot/pkg/controller"
	"cryptoBot/pkg/cron"
	"cryptoBot/pkg/log"
	"cryptoBot/pkg/repository"
	"cryptoBot/pkg/repository/postgres"
	"cryptoBot/pkg/service/date"
	"cryptoBot/pkg/service/exchange"
	"cryptoBot/pkg/service/indicator"
	"cryptoBot/pkg/service/telegram"
	"cryptoBot/pkg/service/trading"
	"fmt"
	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
	migrate "github.com/rubenv/sql-migrate"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"os"
	"os/signal"
	"strconv"
	"syscall"
)

func main() {
	if err := godotenv.Load(); err != nil {
		panic(fmt.Sprintf("Failed load env file: %s", err.Error()))
	}
	if err := initConfig(); err != nil {
		panic(fmt.Sprintf("Error during reading configs: %s", err.Error()))
	}
	configs.NewRuntimeConfig()

	log.InitLogger()

	var closableClosure []func()

	defer func() {
		for i := range closableClosure {
			closableClosure[i]()
		}
	}()

	zap.S().Info("Trading bot is starting...\n")

	postgresDbPort, _ := strconv.ParseInt(os.Getenv("DB_PORT_PROD"), 10, 64)
	postgresDb, err := postgres.NewPostgresDb(&postgres.Config{
		Host:     os.Getenv("DB_HOST"),
		Port:     int(postgresDbPort),
		Username: os.Getenv("DB_USERNAME"),
		Password: os.Getenv("DB_PASSWORD"),
		DBName:   os.Getenv("DB_NAME"),
		SSLMode:  os.Getenv("DB_SSLMODE"),
	})
	if err != nil {
		zap.S().Fatalf("FAILED to init db %s", err.Error())
		return
	}

	closableClosure = append(closableClosure, func() {
		err := postgresDb.Close()
		if err != nil {
			zap.S().Errorf("Error during closing postgres connection: %s", err.Error())
		}
	})

	initMigrations(postgresDb)

	repos := repository.NewRepositories(postgresDb)

	exchangeApi := bybit.NewBybitApi(os.Getenv("BYBIT_CryptoBotTrendScalper_API_KEY"), os.Getenv("BYBIT_CryptoBotTrendScalper_API_SECRET"))

	maService := indicator.NewMovingAverageService(date.GetClock(), repos.Kline)
	stdDevService := indicator.NewStandardDeviationService(date.GetClock(), repos.Kline)
	exchangeDataService := exchange.NewExchangeDataService(repos.Transaction, repos.Coin, exchangeApi, date.GetClock(), repos.Kline)
	priceChangeTrackingService := trading.NewPriceChangeTrackingService(repos.PriceChange)
	fetcherService := exchange.NewKlinesFetcherService(exchangeApi, repos.Kline)

	maTradingService := trading.NewMAStrategyTradingService(repos.Transaction, repos.PriceChange, exchangeApi, date.GetClock(), exchangeDataService, repos.Kline, priceChangeTrackingService, maService, stdDevService, fetcherService)

	telegramService := telegram.NewTelegramService(repos.Transaction, repos.Coin, exchangeApi)

	if enabled, err := strconv.ParseBool(os.Getenv("TRADING_ENABLED")); enabled && err == nil {
		cron.InitCronJobs(maTradingService, repos.Coin)
	}

	router := controller.InitControllers(telegramService)

	srv := new(cryptoBot.Server)
	go func() {
		zap.S().Info("Server is doing to be up right now!\n")
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

func initConfig() error {
	viper.AddConfigPath("configs")
	viper.SetConfigName("config")
	return viper.ReadInConfig()
}

func initMigrations(db *sqlx.DB) {
	migrations := &migrate.FileMigrationSource{
		Dir: "./migrations",
	}

	n, err := migrate.Exec(db.DB, "postgres", migrations, migrate.Up)
	if err != nil {
		zap.S().Errorf("Error during applying migrations! %s", err.Error())
	}
	zap.S().Infof("Applied %d migrations!\n", n)
}
