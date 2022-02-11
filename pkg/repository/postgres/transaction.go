package postgres

import (
	"cryptoBot/pkg/constants"
	"cryptoBot/pkg/data/domains"
	"fmt"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
	"strings"
	"time"
)

func NewTransaction(db *sqlx.DB) *Transaction {
	return &Transaction{db: db}
}

type Transaction struct {
	db *sqlx.DB
}

func (r *Transaction) FindLastByCoinId(coinId int64) (*domains.Transaction, error) {
	var transaction domains.Transaction
	if err := r.db.Get(&transaction, "SELECT * FROM transaction_table WHERE coin_id=$1 order by created_at desc limit 1", int64(coinId)); err != nil {
		if strings.Contains(err.Error(), "no rows in result set") {
			return nil, nil
		}
		return nil, err
	}
	return &transaction, nil
}

func (r *Transaction) FindLastByCoinIdAndType(coinId int64, transactionType constants.TransactionType) (*domains.Transaction, error) {
	var transaction domains.Transaction
	if err := r.db.Get(&transaction, "SELECT * FROM transaction_table WHERE coin_id=$1 and transaction_type=$2 order by created_at desc limit 1", int64(coinId), transactionType); err != nil {
		if strings.Contains(err.Error(), "no rows in result set") {
			return nil, nil
		}
		return nil, err
	}
	return &transaction, nil
}

func (r *Transaction) FindLastBoughtNotSold(coinId int64) (*domains.Transaction, error) {
	var transaction domains.Transaction
	if err := r.db.Get(&transaction, "SELECT * FROM transaction_table WHERE coin_id=$1 and transaction_type=$2 and related_transaction_id is null order by created_at desc limit 1", int64(coinId), constants.BUY); err != nil {
		if strings.Contains(err.Error(), "no rows in result set") {
			return nil, nil
		}
		return nil, err
	}
	return &transaction, nil
}

func (r *Transaction) FindLastBoughtNotSoldAndDate(date time.Time) (*domains.Transaction, error) {
	var transaction domains.Transaction
	if err := r.db.Get(&transaction, "SELECT * FROM transaction_table WHERE transaction_type=$1 and related_transaction_id is null and date_trunc('day', created_at) = $2 order by created_at desc limit 1", constants.BUY, date); err != nil {
		if strings.Contains(err.Error(), "no rows in result set") {
			return nil, nil
		}
		return nil, err
	}
	return &transaction, nil
}

func (r *Transaction) CalculateSumOfProfit() (int64, error) {
	var sumOfProfit int64
	err := r.db.Get(&sumOfProfit, "select sum(profit) from transaction_table where profit is not null")
	return sumOfProfit, err
}

func (r *Transaction) CalculateSumOfSpentTransactions() (int64, error) {
	var sumOfSpent int64
	err := r.db.Get(&sumOfSpent, "select sum(total_cost) from transaction_table where related_transaction_id is null")
	return sumOfSpent, err
}

func (r *Transaction) CalculateSumOfSpentTransactionsAndCreatedAfter(date time.Time) (int64, error) {
	var sumOfSpent int64
	err := r.db.Get(&sumOfSpent, "select sum(total_cost) from transaction_table where related_transaction_id is null and created_at > $1", date)
	return sumOfSpent, err
}

func (r *Transaction) CalculateSumOfProfitByDate(date time.Time) (int64, error) {
	var sumOfProfit int64
	err := r.db.Get(&sumOfProfit, "select sum(profit) from transaction_table where profit is not null and date_trunc('day', created_at) = $1", date)
	return sumOfProfit, err
}

func (r *Transaction) FindMinPriceByDate(date time.Time) (int64, error) {
	var sumOfSpent int64
	err := r.db.Get(&sumOfSpent, "select min(price) from transaction_table where date_trunc('day', created_at) = $1", date)
	return sumOfSpent, err
}

func (r *Transaction) CalculateSumOfSpentTransactionsByDate(date time.Time) (int64, error) {
	var sumOfSpent int64
	err := r.db.Get(&sumOfSpent, "select sum(total_cost) from transaction_table where related_transaction_id is null and date_trunc('day', created_at) = $1", date)
	return sumOfSpent, err
}

func (r *Transaction) CalculateSumOfTransactionsByDateAndType(date time.Time, transType constants.TransactionType) (int64, error) {
	var sumOfSpent int64
	err := r.db.Get(&sumOfSpent, "select sum(total_cost) from transaction_table where date_trunc('day', created_at) = $1 and transaction_type = $2", date, transType)
	return sumOfSpent, err
}

func (r *Transaction) SaveTransaction(trnsctn *domains.Transaction) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}

	utc, _ := time.LoadLocation("UTC")
	if trnsctn.Id == 0 {
		transactionId := int64(0)
		err := tx.QueryRow("INSERT INTO transaction_table (coin_id, transaction_type, amount, price, total_cost, created_at, client_order_id, api_error, related_transaction_id, profit, percent_profit, commission) values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12) RETURNING id",
			trnsctn.CoinId, trnsctn.TransactionType, trnsctn.Amount, trnsctn.Price, trnsctn.TotalCost, time.Now().In(utc), trnsctn.ClientOrderId, trnsctn.ApiError, trnsctn.RelatedTransactionId, trnsctn.Profit, trnsctn.PercentProfit, trnsctn.Commission,
		).Scan(&transactionId)
		if err != nil {
			_ = tx.Rollback()
			zap.S().Errorf("Invalid try to save Domain on proxy side: %s. "+
				"Error: %s", trnsctn.String(), err.Error())
			return err
		}
		trnsctn.Id = transactionId
		zap.S().Debugf("Domain was saved on proxy side: %s", trnsctn.String())
		return tx.Commit()
	}

	resp, err := tx.Exec("UPDATE transaction_table SET coin_id = $2, transaction_type = $3, amount = $4, price = $5, total_cost = $6, client_order_id = $7, api_error = $8, related_transaction_id = $9, profit = $10, percent_profit = $11, commission = $12 WHERE id = $1",
		trnsctn.Id, trnsctn.CoinId, trnsctn.TransactionType, trnsctn.Amount, trnsctn.Price, trnsctn.TotalCost, trnsctn.ClientOrderId, trnsctn.ApiError, trnsctn.RelatedTransactionId, trnsctn.Profit, trnsctn.PercentProfit, trnsctn.Commission)
	if err != nil {
		_ = tx.Rollback()
		zap.S().Errorf("Invalid try to update domain on proxy side: %s. "+
			"Error: %s", trnsctn.String(), err.Error())
		return err
	}

	if count, err := resp.RowsAffected(); err != nil {
		_ = tx.Rollback()
		return err
	} else if count != 1 {
		_ = tx.Rollback()
		return fmt.Errorf("Unexpected updated rows count: %d", count)
	}

	zap.S().Infof("Domain was updated on proxy side: %s", trnsctn.String())
	return tx.Commit()
}
