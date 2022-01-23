package repository

import (
	"cryptoBot/pkg/constants"
	"cryptoBot/pkg/data/domains"
	"cryptoBot/pkg/repository/postgres"
	"github.com/jmoiron/sqlx"
	"time"
)

type Coin interface {
	FindBySymbol(symbol string) (*domains.Coin, error)
}

type Transaction interface {
	FindLastByCoinId(coinId int64) (*domains.Transaction, error)
	FindLastByCoinIdAndType(coinId int64, transactionType constants.TransactionType) (*domains.Transaction, error)
	FindLastBoughtNotSold(coinId int64) (*domains.Transaction, error)
	FindLastBoughtNotSoldAndDate(date time.Time) (*domains.Transaction, error)
	SaveTransaction(transaction *domains.Transaction) error
	CalculateSumOfProfit() (int64, error)
	CalculateSumOfSpentTransactions() (int64, error)
	CalculateSumOfProfitByDate(date time.Time) (int64, error)
	FindMinPriceByDate(date time.Time) (int64, error)
	CalculateSumOfSpentTransactionsByDate(date time.Time) (int64, error)
	CalculateSumOfTransactionsByDateAndType(date time.Time, transType constants.TransactionType) (int64, error)
}

type PriceChange interface {
	FindByTransactionId(transactionId int64) (*domains.PriceChange, error)
	SavePriceChange(priceChange *domains.PriceChange) error
}

type Repository struct {
	Coin        Coin
	Transaction Transaction
	PriceChange PriceChange
}

func NewRepositories(postgresDb *sqlx.DB) *Repository {
	return &Repository{
		Coin:        postgres.NewCoin(postgresDb),
		Transaction: postgres.NewTransaction(postgresDb),
		PriceChange: postgres.NewPriceChange(postgresDb),
	}
}
