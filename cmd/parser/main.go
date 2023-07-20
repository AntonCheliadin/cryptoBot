package main

import (
	"cryptoBot/cmd/bootstrap"
	"cryptoBot/pkg/constants"
	"cryptoBot/pkg/log"
	"cryptoBot/pkg/repository"
	"cryptoBot/pkg/service/parser"
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

	var closableClosure []func()

	defer func() {
		for i := range closableClosure {
			closableClosure[i]()
		}
	}()

	zap.S().Info("Trading bot is starting...")

	postgresDb := bootstrap.Database(closableClosure)

	closableClosure = append(closableClosure, func() {
		err := postgresDb.Close()
		if err != nil {
			zap.S().Errorf("Error during closing postgres connection: %s", err.Error())
		}
	})

	repos := repository.NewRepositories(postgresDb)
	parserService := parser.NewBybitArchiveParseService(repos.Kline)

	//downloadCoinHistory(repos, parserService, "MATICUSDT", "2023-06-15", "2023-07-14")
	//downloadCoinHistory(repos, parserService, "UNIUSDT", "2023-06-15", "2023-07-14")
	//downloadCoinHistory(repos, parserService, "NEARUSDT", "2023-06-15", "2023-07-14")
	//downloadCoinHistory(repos, parserService, "IMXUSDT", "2023-06-15", "2023-07-14")
	//downloadCoinHistory(repos, parserService, "FLOWUSDT", "2023-06-15", "2023-07-14")
	//downloadCoinHistory(repos, parserService, "FILUSDT", "2023-06-15", "2023-07-14")
	//downloadCoinHistory(repos, parserService, "DYDXUSDT", "2023-06-15", "2023-07-14")
	//downloadCoinHistory(repos, parserService, "DASHUSDT", "2023-06-15", "2023-07-14")
	//downloadCoinHistory(repos, parserService, "ALGOUSDT", "2023-06-15", "2023-07-14")
	//downloadCoinHistory(repos, parserService, "ZECUSDT", "2023-06-15", "2023-07-14")
	//downloadCoinHistory(repos, parserService, "SOLUSDT", "2023-06-15", "2023-07-14")
	downloadCoinHistory(repos, parserService, "XRPUSDT", "2023-06-01", "2023-07-14")
	downloadCoinHistory(repos, parserService, "LTCUSDT", "2023-06-01", "2023-07-14")

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
