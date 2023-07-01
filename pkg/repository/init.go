package repository

import (
	"cryptoBot/pkg/constants"
	"cryptoBot/pkg/data/domains"
	"cryptoBot/pkg/data/dto/postgres/transaction"
	"cryptoBot/pkg/repository/postgres"
	"github.com/jmoiron/sqlx"
	"time"
)

type Coin interface {
	FindBySymbol(symbol string) (*domains.Coin, error)
	FindById(id int64) (*domains.Coin, error)
}

type Transaction interface {
	FindById(id int64) (*domains.Transaction, error)
	FindLastByCoinId(coinId int64, tradingStrategy constants.TradingStrategy) (*domains.Transaction, error)
	FindLastByCoinIdAndType(coinId int64, transactionType constants.TransactionType, tradingStrategy constants.TradingStrategy) (*domains.Transaction, error)
	FindLastBoughtNotSold(coinId int64, tradingStrategy constants.TradingStrategy) (*domains.Transaction, error)
	FindLastBoughtNotSoldAndDate(date time.Time, tradingStrategy constants.TradingStrategy) (*domains.Transaction, error)
	SaveTransaction(transaction *domains.Transaction) error
	CalculateSumOfProfit(tradingStrategy constants.TradingStrategy) (int64, error)
	CalculateSumOfProfitByCoin(coinId int64, tradingStrategy constants.TradingStrategy) (int64, error)
	CalculateSumOfSpentTransactions(tradingStrategy constants.TradingStrategy) (int64, error)
	CalculateSumOfSpentTransactionsAndCreatedAfter(date time.Time, tradingStrategy constants.TradingStrategy) (int64, error)
	CalculateSumOfProfitByDate(date time.Time, tradingStrategy constants.TradingStrategy) (int64, error)
	FindMinPriceByDate(date time.Time, tradingStrategy constants.TradingStrategy) (int64, error)
	CalculateSumOfSpentTransactionsByDate(date time.Time, tradingStrategy constants.TradingStrategy) (int64, error)
	CalculateSumOfTransactionsByDateAndType(date time.Time, transType constants.TransactionType, tradingStrategy constants.TradingStrategy) (int64, error)

	FindOpenedTransaction(tradingStrategy constants.TradingStrategy) (*domains.Transaction, error)
	FindAllOpenedTransactions(tradingStrategy constants.TradingStrategy) ([]*domains.Transaction, error)
	FindOpenedTransactionByCoin(tradingStrategy constants.TradingStrategy, coinId int64) (*domains.Transaction, error)

	FindAllProfitPercents(tradingStrategy int) ([]transaction.TransactionProfitPercentsDto, error)
	FindAllCoinIds(tradingStrategy int) ([]int64, error)
}

type ConditionalOrder interface {
	FindByTransaction(transaction *domains.Transaction) (*domains.ConditionalOrder, error)
}

type PriceChange interface {
	FindByTransactionId(transactionId int64) (*domains.PriceChange, error)
	SavePriceChange(priceChange *domains.PriceChange) error
}

type Kline interface {
	FindAllByCoinIdAndIntervalAndCloseTimeLessOrderByOpenTimeWithLimit(coinId int64, interval string, closeTime time.Time, limit int64) ([]*domains.Kline, error)
	FindAllByCoinIdAndIntervalAndCloseTimeInRange(coinId int64, interval string, openTime time.Time, closeTime time.Time) ([]*domains.Kline, error)
	SaveKline(domain *domains.Kline) error
	FindOpenedAtMoment(coinId int64, momentTime time.Time, interval string) (*domains.Kline, error)
	FindClosedAtMoment(coinId int64, momentTime time.Time, interval string) (*domains.Kline, error)
	FindLast(coinId int64, interval string) (*domains.Kline, error)
}

type SyntheticKline interface {
	FindAllByCoinIdsAndIntervalAndCloseTimeInRange(coinId1 int64, coinId2 int64, interval string, openTime time.Time, closeTime time.Time) ([]domains.IKline, error)
	FindAllByCoinIdAndIntervalAndCloseTimeLessOrderByOpenTimeWithLimit(coinId1 int64, coinId2 int64, interval string, closeTime time.Time, limit int) ([]domains.IKline, error)
	RefreshView() error
}

type Repository struct {
	Coin             Coin
	Transaction      Transaction
	PriceChange      PriceChange
	Kline            Kline
	ConditionalOrder ConditionalOrder
	SyntheticKline   SyntheticKline
}

func NewRepositories(postgresDb *sqlx.DB) *Repository {
	return &Repository{
		Coin:             postgres.NewCoin(postgresDb),
		Transaction:      postgres.NewTransaction(postgresDb),
		PriceChange:      postgres.NewPriceChange(postgresDb),
		Kline:            postgres.NewKline(postgresDb),
		ConditionalOrder: postgres.NewConditionalOrder(postgresDb),
		SyntheticKline:   postgres.NewSyntheticKline(postgresDb),
	}
}
