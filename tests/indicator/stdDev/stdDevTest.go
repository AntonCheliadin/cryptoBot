package main

import (
	"cryptoBot/pkg/log"
	"cryptoBot/pkg/repository"
	"cryptoBot/pkg/repository/postgres"
	"cryptoBot/pkg/service/date"
	"cryptoBot/pkg/service/indicator"
	"cryptoBot/pkg/service/indicator/techanLib"
	"fmt"
	"github.com/joho/godotenv"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"os"
	"strconv"
	"time"
)

func initLocalConfig() error {
	viper.AddConfigPath("configs")
	viper.SetConfigName("config")
	return viper.ReadInConfig()
}

func main() {
	if err := godotenv.Load(); err != nil {
		panic(fmt.Sprintf("Failed load env file: %s", err.Error()))
	}
	if err := initLocalConfig(); err != nil {
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

	repos := repository.NewRepositories(postgresDb)

	test(repos)

	if err := postgresDb.Close(); err != nil {
		zap.S().Errorf("error occured on db connection close: %s", err.Error())
	}

	os.Exit(0)
}

func test(repos *repository.Repository) {
	timeIterator := time.Now() //time.Parse(constants.DATE_TIME_FORMAT, "2022-09-07 21:47:01")
	seriesConvertorService := techanLib.NewTechanConvertorService(date.GetClockMock(timeIterator), repos.Kline)
	indicatorService := indicator.NewStandardDeviationService(date.GetClockMock(timeIterator), repos.Kline, seriesConvertorService)

	coin, _ := repos.Coin.FindBySymbol("SOLUSDT")

	value := indicatorService.CalculateCurrentStandardDeviationForPriceChange(coin, "15")
	zap.S().Infof("value=%v", value)
}
