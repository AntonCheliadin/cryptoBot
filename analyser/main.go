package main

import (
	"cryptoBot/pkg/api/bybit/mock"
	"cryptoBot/pkg/log"
	"cryptoBot/pkg/repository"
	"cryptoBot/pkg/repository/postgres"
	"cryptoBot/pkg/service/analyser"
	"cryptoBot/pkg/service/date"
	"cryptoBot/pkg/service/exchange"
	"cryptoBot/pkg/service/indicator"
	"cryptoBot/pkg/service/trading"
	"fmt"
	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
	migrate "github.com/rubenv/sql-migrate"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"os"
	"strconv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		panic(fmt.Sprintf("Failed load env file: %s", err.Error()))
	}
	if err := initConfig(); err != nil {
		panic(fmt.Sprintf("Error during reading configs: %s", err.Error()))
	}

	log.InitLoggerAnalyser()

	var closableClosure []func()

	defer func() {
		for i := range closableClosure {
			closableClosure[i]()
		}
	}()

	zap.S().Info("Trading bot is starting...\n")

	postgresDbPort, _ := strconv.ParseInt(os.Getenv("DB_PORT"), 10, 64)
	postgresDb, err := postgres.NewPostgresDb(&postgres.Config{
		Host:     os.Getenv("DB_HOST"),
		Port:     int(postgresDbPort),
		Username: os.Getenv("DB_USERNAME"),
		Password: os.Getenv("DB_PASSWORD"),
		DBName:   os.Getenv("DB_ANALYSER_NAME"),
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

	//exchangeApi := binance.NewBinanceApi()
	//mockExchangeApi := mock.NewBinanceApiMock()

	//exchangeApi := bybit.NewBybitApi()
	mockExchangeApi := mock.NewBybitApiMock()

	//tradingService := trading.NewHolderStrategyTradingService(repos.Transaction, repos.PriceChange, mockExchangeApi)
	//analyserService := analyser.NewAnalyserService(repos.Transaction, repos.PriceChange, exchangeApi, tradingService)
	maService := indicator.NewMovingAverageService(date.GetClock(), repos.Kline)
	exchangeDataService := exchange.NewExchangeDataService(repos.Transaction, repos.Coin, mockExchangeApi, date.GetClock(), repos.Kline)
	priceChangeTrackingService := trading.NewPriceChangeTrackingService(repos.PriceChange)
	maTradingService := trading.NewMAStrategyTradingService(repos.Transaction, repos.PriceChange, mockExchangeApi, date.GetClock(), exchangeDataService, repos.Kline, priceChangeTrackingService, maService)
	analyserService := analyser.NewMovingAverageStrategyAnalyserService(repos.Transaction, repos.PriceChange, mockExchangeApi, maTradingService, repos.Kline)

	coin, _ := repos.Coin.FindBySymbol("SOLUSDT")

	//analyserService.FetchKlines(coin, "2022-03-01", "2022-03-15")
	analyserService.AnalyseCoin(coin, "2022-03-03", "2022-05-23") //max interval  2022-03-03 2022-05-23

	if err := postgresDb.Close(); err != nil {
		zap.S().Errorf("error occured on db connection close: %s", err.Error())
	}

	os.Exit(0)
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
