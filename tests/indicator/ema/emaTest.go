package main

import (
	"cryptoBot/pkg/constants"
	constantIndicator "cryptoBot/pkg/constants/indicator"
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

	repos := repository.NewRepositories(postgresDb)

	testEMA(repos)

	if err := postgresDb.Close(); err != nil {
		zap.S().Errorf("error occured on db connection close: %s", err.Error())
	}

	os.Exit(0)
}

func testEMA(repos *repository.Repository) {
	timeIterator, _ := time.Parse(constants.DATE_TIME_FORMAT, "2022-08-28 03:00:01")
	timeMax, _ := time.Parse(constants.DATE_TIME_FORMAT, "2022-08-28 11:15:01")

	seriesConvertorService := techanLib.NewTechanConvertorService(date.GetClockMock(timeIterator), repos.Kline)
	maIndicatorService := indicator.NewExponentialMovingAverageService(seriesConvertorService)

	coin, _ := repos.Coin.FindBySymbol("SOLUSDT")

	for ; timeIterator.Before(timeMax); timeIterator = timeIterator.Add(time.Minute * 15) {
		seriesConvertorService.Clock = date.GetClockMock(timeIterator)

		//emaResult5 := maIndicatorService.CalculateEMA(coin, "15", 5)
		//emaResult11 := maIndicatorService.CalculateEMA(coin, "15", 11)
		//zap.S().Infof("emaResult5=%v <%v> emaResult11=%v at %v", emaResult5, emaResult5.GTE(emaResult11), emaResult11, timeIterator)

		signal1 := maIndicatorService.IsFastEmaAbove(coin, "15", 5, constantIndicator.EMA, 11, constantIndicator.EMA)
		signal2 := maIndicatorService.IsFastEmaAbove(coin, "15", 13, constantIndicator.EMA, 36, constantIndicator.SMA)
		zap.S().Infof("signal1=%v signal2=%v at %v", signal1, signal2, timeIterator)
	}
}
