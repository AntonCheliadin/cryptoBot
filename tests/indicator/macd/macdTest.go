package main

import (
	"cryptoBot/pkg/constants"
	"cryptoBot/pkg/log"
	"cryptoBot/pkg/repository"
	"cryptoBot/pkg/repository/postgres"
	"cryptoBot/pkg/service/date"
	"cryptoBot/pkg/service/indicator"
	"cryptoBot/pkg/service/indicator/techanLib"
	"fmt"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
	"os"
	"strconv"
	"time"
)

func main() {
	if err := godotenv.Load(); err != nil {
		panic(fmt.Sprintf("Failed load env file: %s", err.Error()))
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

	testMACD(repos)

	if err := postgresDb.Close(); err != nil {
		zap.S().Errorf("error occured on db connection close: %s", err.Error())
	}

	os.Exit(0)
}

func testMACD(repos *repository.Repository) {
	timeMock, _ := time.Parse(constants.DATE_TIME_FORMAT, "2022-08-28 12:03:01")
	seriesConvertorService := techanLib.NewTechanConvertorService(date.GetClockMock(timeMock), repos.Kline)
	macdIndicatorService := indicator.NewMACDService(seriesConvertorService)

	coin, _ := repos.Coin.FindBySymbol("SOLUSDT")

	macdResult := macdIndicatorService.CalculateCurrentMACD(coin, "15", 8, 21, 5)
	zap.S().Infof("MACD=%v at %v", macdResult, timeMock)
}
