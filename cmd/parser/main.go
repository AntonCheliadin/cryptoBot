package main

import (
	"cryptoBot/pkg/constants"
	"cryptoBot/pkg/log"
	"cryptoBot/pkg/repository"
	"cryptoBot/pkg/repository/postgres"
	"cryptoBot/pkg/service/parser"
	"fmt"
	"github.com/joho/godotenv"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"os"
	"strconv"
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

	var closableClosure []func()

	defer func() {
		for i := range closableClosure {
			closableClosure[i]()
		}
	}()

	zap.S().Info("Trading bot is starting...")

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

	repos := repository.NewRepositories(postgresDb)
	parserService := parser.NewBybitArchiveParseService(repos.Kline)

	downloadCoinHistory(repos, parserService, "MATICUSDT", "2021-06-29", "2023-06-16")
	//downloadCoinHistory(repos, parserService, "UNIUSDT", "2021-03-18", "2023-06-16")
	//downloadCoinHistory(repos, parserService, "NEARUSDT", "2021-10-11", "2023-06-16")
	//downloadCoinHistory(repos, parserService, "IMXUSDT", "2021-11-24", "2023-06-16")
	//downloadCoinHistory(repos, parserService, "FLOWUSDT", "2021-11-26", "2023-06-16")
	//downloadCoinHistory(repos, parserService, "FILUSDT", "2021-06-29", "2023-06-16")
	//downloadCoinHistory(repos, parserService, "DYDXUSDT", "2021-10-11", "2023-06-16")
	//downloadCoinHistory(repos, parserService, "DASHUSDT", "2021-10-12", "2023-06-16")
	//downloadCoinHistory(repos, parserService, "ALGOUSDT", "2021-09-23", "2023-06-16")
	//downloadCoinHistory(repos, parserService, "ZECUSDT", "2021-11-24", "2023-06-16")
	//downloadCoinHistory(repos, parserService, "SOLUSDT", "2023-01-21", "2023-06-16")

	if err := postgresDb.Close(); err != nil {
		zap.S().Errorf("error occured on db connection close: %s", err.Error())
	}

	os.Exit(0)
}

func downloadCoinHistory(repos *repository.Repository, parserService *parser.BybitArchiveParseService, symbol string, from string, to string) {
	coin, _ := repos.Coin.FindBySymbol(symbol)

	timeFrom, _ := time.Parse(constants.DATE_FORMAT, from)
	timeTo, _ := time.Parse(constants.DATE_FORMAT, to)

	if err := parserService.Parse(coin, timeFrom, timeTo, 60); err != nil {
		zap.S().Errorf("Error during parse %s", err.Error())
	}
}

func initConfig() error {
	viper.AddConfigPath("configs")
	viper.SetConfigName("config")
	return viper.ReadInConfig()
}
