package main

import (
	"context"
	"cryptoBot"
	"cryptoBot/pkg/api/binance"
	"cryptoBot/pkg/controller"
	"cryptoBot/pkg/cron"
	"cryptoBot/pkg/log"
	"cryptoBot/pkg/repository"
	"cryptoBot/pkg/repository/postgres"
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

	log.InitLogger()

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

	exchangeApi := binance.NewBinanceApi()

	tradingService := trading.NewTradingService(repos.Transaction, repos.PriceChange, exchangeApi)
	telegramService := telegram.NewTelegramService(repos.Transaction)

	cron.InitCronJobs(tradingService, repos.Coin)

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
