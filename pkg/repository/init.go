package repository

import (
	"github.com/jmoiron/sqlx"
	"tradingBot/pkg/constants"
	"tradingBot/pkg/data/domains"
	"tradingBot/pkg/repository/postgres"
)

type Coin interface {
	FindBySymbol(symbol string) (*domains.Coin, error)
}

type Transaction interface {
	FindLastByCoinId(coinId int64) (*domains.Transaction, error)
	FindLastByCoinIdAndType(coinId int64, transactionType constants.TransactionType) (*domains.Transaction, error)
	FindLastBoughtNotSold(coinId int64) (*domains.Transaction, error)
	SaveTransaction(transaction *domains.Transaction) error
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
