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
	"github.com/sdcoffey/big"
	"github.com/sdcoffey/techan"
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

	testRSI(repos)
	//TestRelativeStrengthIndicatorNoPriceChange(5)
	//TestRelativeStrengthIndicatorNoPriceChange(13)

	if err := postgresDb.Close(); err != nil {
		zap.S().Errorf("error occured on db connection close: %s", err.Error())
	}

	os.Exit(0)
}

func testRSI(repos *repository.Repository) {
	timeMock, _ := time.Parse(constants.DATE_TIME_FORMAT, "2022-08-28 11:03:01")
	seriesConvertorService := techanLib.NewTechanConvertorService(date.NewClockMock(timeMock), repos.Kline)
	rsiIndicatorService := indicator.NewRelativeStrengthIndexService(seriesConvertorService)

	coin, _ := repos.Coin.FindBySymbol("SOLUSDT")

	rsi5 := rsiIndicatorService.CalculateCurrentRSI(coin, "15", 5)
	zap.S().Infof("RSI=%v at %v with length=%v", rsi5, timeMock, 5)

	rsi13 := rsiIndicatorService.CalculateCurrentRSI(coin, "15", 13)
	zap.S().Infof("RSI=%v at %v with length=%v ", rsi13, timeMock, 13)
}

func TestRelativeStrengthIndicatorNoPriceChange(timeFrame int) {
	series := mockTimeSeries("36.33", "36.62", "36.9", "36.78", "36.82", "36.1", "36.36", "36.6", "36.53", "36.59", "36.53", "36.49", "36.42", "36.17", "36.3")
	close := techan.NewClosePriceIndicator(series)
	rsInd := techan.NewRelativeStrengthIndexIndicator(close, timeFrame)

	rsi := make([]string, len(series.Candles))

	for i := 0; i < len(series.Candles); i++ {
		rsi[i] = rsInd.Calculate(i).String()
	}
	//rsi := rsInd.Calculate(5)

	zap.S().Infof("RSI=%v for %v timeFrame", rsi, timeFrame)
}

func mockTimeSeries(values ...string) *techan.TimeSeries {
	var candleIndex int

	ts := techan.NewTimeSeries()
	for i, val := range values {
		if i < len(values)-1 {
			candle := techan.NewCandle(techan.NewTimePeriod(time.Unix(int64(candleIndex), 0), time.Second))
			candle.OpenPrice = big.NewFromString(val)
			candle.ClosePrice = big.NewFromString(values[i+1])
			candle.MaxPrice = big.NewFromString(val).Add(big.ONE)
			candle.MinPrice = big.NewFromString(val).Sub(big.ONE)
			candle.Volume = big.NewFromString(val)

			ts.AddCandle(candle)

			candleIndex++
		}
	}

	return ts
}
