package main

import (
	"cryptoBot/pkg/constants"
	"cryptoBot/pkg/log"
	"cryptoBot/pkg/repository"
	"cryptoBot/pkg/repository/postgres"
	"cryptoBot/pkg/util"
	"fmt"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
	"os"
	"strconv"
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

	testStatistic(repos)

	if err := postgresDb.Close(); err != nil {
		zap.S().Errorf("error occured on db connection close: %s", err.Error())
	}

	os.Exit(0)
}

func testStatistic(repos *repository.Repository) {

	ids := []int64{5, 6}

	rows, err := repos.Transaction.FetchStatisticByDays(int(constants.PAIR_ARBITRAGE), ids)

	if err != nil {
		zap.S().Errorf("error %s", err.Error())
		return
	}

	zap.S().Infof(fmt.Sprintf("| %v | %10v | %10v | %10v |",
		"date", "profit", "%", "size"))
	for k := 0; k < len(rows); k += 1 {
		dto := rows[k]
		zap.S().Infof(fmt.Sprintf("\n| %v | %10v | %10v | %10v |",
			dto.CreatedAt, util.GetDollarsByCents(dto.ProfitInCents), dto.ProfitPercent, dto.OrdersSize))
	}

}
