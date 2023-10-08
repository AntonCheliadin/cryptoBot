package postgres

import (
	"cryptoBot/pkg/constants"
	"cryptoBot/pkg/data/domains"
	"cryptoBot/pkg/data/dto/postgres/transaction"
	"database/sql"
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

func (r *Transaction) find(query string, args ...interface{}) (*domains.Transaction, error) {
	var transaction domains.Transaction
	if err := r.db.Get(&transaction, query, args); err != nil {
		if strings.Contains(err.Error(), "no rows in result set") {
			return nil, nil
		}
		return nil, err
	}
	return &transaction, nil
}

func (r *Transaction) FindOpenedTransaction(tradingStrategy constants.TradingStrategy) (*domains.Transaction, error) {
	var transaction domains.Transaction
	if err := r.db.Get(&transaction, "SELECT * FROM transaction_table WHERE related_transaction_id is null AND trading_strategy=$1 order by created_at desc limit 1", tradingStrategy); err != nil {
		if strings.Contains(err.Error(), "no rows in result set") {
			return nil, nil
		}
		return nil, err
	}
	return &transaction, nil
}

func (r *Transaction) FindOpenedTransactionByCoin(tradingStrategy constants.TradingStrategy, coinId int64) (*domains.Transaction, error) {
	var transaction domains.Transaction
	if err := r.db.Get(&transaction, "SELECT * FROM transaction_table WHERE related_transaction_id is null AND trading_strategy=$1 AND coin_id=$2 order by created_at desc limit 1", tradingStrategy, coinId); err != nil {
		if strings.Contains(err.Error(), "no rows in result set") {
			return nil, nil
		}
		return nil, err
	}
	return &transaction, nil
}

func (r *Transaction) FindAllOpenedTransactions(tradingStrategy constants.TradingStrategy) ([]*domains.Transaction, error) {
	var klines []domains.Transaction
	err := r.db.Select(&klines, "SELECT * FROM transaction_table WHERE related_transaction_id is null AND trading_strategy=$1 order by created_at desc",
		tradingStrategy)

	if err != nil {
		return nil, fmt.Errorf("Error during select domain: %s", err.Error())
	}

	return r.listRelationsToListRelationsPointers(klines), nil
}

func (r *Transaction) FindAllProfitPercents(tradingStrategy int) ([]transaction.TransactionProfitPercentsDto, error) {
	var profitPercents []transaction.TransactionProfitPercentsDto
	err := r.db.Select(&profitPercents, "select created_at, sum(percent_profit) profit_percent from transaction_table where trading_strategy = $1 and profit is not null group by created_at order by created_at asc;",
		tradingStrategy)

	if err != nil {
		return nil, fmt.Errorf("Error during select domain: %s", err.Error())
	}

	return profitPercents, nil
}

func (r *Transaction) FetchStatisticByDays(tradingStrategy int, coinIds []int64) ([]transaction.PairTransactionProfitPercentsDto, error) {
	var profitPercents []transaction.PairTransactionProfitPercentsDto

	selectQuery := "select to_char(created_at, 'YYYY-MM-DD') created_date, avg(percent_profit) profit_percent_of_paired_order, sum(profit) profit_sum, count(1) / 2 orders_size from transaction_table where trading_strategy = ?  and profit is not null    and coin_id in (?) group by to_char(created_at, 'YYYY-MM-DD') order by to_char(created_at, 'YYYY-MM-DD') desc limit 3;"
	preparedQuery, preparedParameters, _ := sqlx.In(selectQuery, tradingStrategy, coinIds)
	err := r.db.Select(&profitPercents, r.db.Rebind(preparedQuery), preparedParameters...)

	if err != nil {
		return nil, fmt.Errorf("Error during select domain: %s", err.Error())
	}

	return profitPercents, nil
}

func (r *Transaction) FindAllCoinIds(tradingStrategy int) ([]int64, error) {
	var results []int64
	err := r.db.Select(&results, "select distinct coin_id from transaction_table where trading_strategy = $1;",
		tradingStrategy)

	if err != nil {
		return nil, fmt.Errorf("Error during select domain: %s", err.Error())
	}

	return results, nil
}

func (r *Transaction) listRelationsToListRelationsPointers(domainList []domains.Transaction) []*domains.Transaction {
	result := make([]*domains.Transaction, 0, len(domainList))
	for i := len(domainList) - 1; i >= 0; i-- {
		result = append(result, &domainList[i])
	}
	return result
}

func (r *Transaction) FindById(id int64) (*domains.Transaction, error) {
	var transaction domains.Transaction
	if err := r.db.Get(&transaction, "SELECT * FROM transaction_table WHERE id=$1", id); err != nil {
		if strings.Contains(err.Error(), "no rows in result set") {
			return nil, nil
		}
		return nil, err
	}
	return &transaction, nil
}

func (r *Transaction) FindLastByCoinId(coinId int64, tradingStrategy constants.TradingStrategy) (*domains.Transaction, error) {
	var transaction domains.Transaction
	if err := r.db.Get(&transaction, "SELECT * FROM transaction_table WHERE coin_id=$1 AND trading_strategy=$2 order by created_at desc limit 1", coinId, tradingStrategy); err != nil {
		if strings.Contains(err.Error(), "no rows in result set") {
			return nil, nil
		}
		return nil, err
	}
	return &transaction, nil
}

func (r *Transaction) FindLastByCoinIdAndType(coinId int64, transactionType constants.TransactionType, tradingStrategy constants.TradingStrategy) (*domains.Transaction, error) {
	var transaction domains.Transaction
	if err := r.db.Get(&transaction, "SELECT * FROM transaction_table WHERE coin_id=$1 and transaction_type=$2 AND trading_strategy=$3 order by created_at desc limit 1", coinId, transactionType, tradingStrategy); err != nil {
		if strings.Contains(err.Error(), "no rows in result set") {
			return nil, nil
		}
		return nil, err
	}
	return &transaction, nil
}

func (r *Transaction) FindLastBoughtNotSold(coinId int64, tradingStrategy constants.TradingStrategy) (*domains.Transaction, error) {
	var transaction domains.Transaction
	if err := r.db.Get(&transaction, "SELECT * FROM transaction_table WHERE coin_id=$1 and transaction_type=$2 and related_transaction_id is null AND trading_strategy=$3 order by created_at desc limit 1", int64(coinId), constants.BUY, tradingStrategy); err != nil {
		if strings.Contains(err.Error(), "no rows in result set") {
			return nil, nil
		}
		return nil, err
	}
	return &transaction, nil
}

func (r *Transaction) FindLastBoughtNotSoldAndDate(date time.Time, tradingStrategy constants.TradingStrategy) (*domains.Transaction, error) {
	var transaction domains.Transaction
	if err := r.db.Get(&transaction, "SELECT * FROM transaction_table WHERE transaction_type=$1 and related_transaction_id is null and date_trunc('day', created_at) = $2 AND trading_strategy=$3 order by created_at desc limit 1", constants.BUY, date, tradingStrategy); err != nil {
		if strings.Contains(err.Error(), "no rows in result set") {
			return nil, nil
		}
		return nil, err
	}
	return &transaction, nil
}

func (r *Transaction) CalculateSumOfProfit(tradingStrategy constants.TradingStrategy) (int64, error) {
	var sumOfProfit int64
	err := r.db.Get(&sumOfProfit, "select sum(profit) from transaction_table where profit is not null AND trading_strategy=$1", tradingStrategy)
	return sumOfProfit, err
}

func (r *Transaction) CalculateSumOfProfitByCoin(coinId int64, tradingStrategy constants.TradingStrategy) (int64, error) {
	var sumOfProfit int64
	err := r.db.Get(&sumOfProfit, "select sum(profit) from transaction_table where profit is not null AND coin_id=$1 AND trading_strategy=$2 AND fake = false", coinId, tradingStrategy)
	return sumOfProfit, err
}

func (r *Transaction) CalculateSumOfSpentTransactions(tradingStrategy constants.TradingStrategy) (int64, error) {
	var sumOfSpent int64
	err := r.db.Get(&sumOfSpent, "select sum(total_cost) from transaction_table where related_transaction_id is null AND trading_strategy=$1", tradingStrategy)
	return sumOfSpent, err
}

func (r *Transaction) CalculateSumOfSpentTransactionsAndCreatedAfter(date time.Time, tradingStrategy constants.TradingStrategy) (int64, error) {
	var sumOfSpent sql.NullInt64
	err := r.db.Get(&sumOfSpent, "select sum(total_cost) from transaction_table where related_transaction_id is null and created_at > $1 AND trading_strategy=$2", date, tradingStrategy)
	return sumOfSpent.Int64, err
}

func (r *Transaction) CalculateSumOfProfitByDate(date time.Time, tradingStrategy constants.TradingStrategy) (int64, error) {
	var sumOfProfit int64
	err := r.db.Get(&sumOfProfit, "select sum(profit) from transaction_table where profit is not null and date_trunc('day', created_at) = $1 AND trading_strategy=$2", date, tradingStrategy)
	return sumOfProfit, err
}

func (r *Transaction) FindMinPriceByDate(date time.Time, tradingStrategy constants.TradingStrategy) (int64, error) {
	var sumOfSpent int64
	err := r.db.Get(&sumOfSpent, "select min(price) from transaction_table where date_trunc('day', created_at) = $1 AND trading_strategy=$2", date, tradingStrategy)
	return sumOfSpent, err
}

func (r *Transaction) CalculateSumOfSpentTransactionsByDate(date time.Time, tradingStrategy constants.TradingStrategy) (int64, error) {
	var sumOfSpent int64
	err := r.db.Get(&sumOfSpent, "select sum(total_cost) from transaction_table where related_transaction_id is null and date_trunc('day', created_at) = $1 AND trading_strategy=$2", date, tradingStrategy)
	return sumOfSpent, err
}

func (r *Transaction) CalculateSumOfTransactionsByDateAndType(date time.Time, transType constants.TransactionType, tradingStrategy constants.TradingStrategy) (int64, error) {
	var sumOfSpent int64
	err := r.db.Get(&sumOfSpent, "select sum(total_cost) from transaction_table where date_trunc('day', created_at) = $1 and transaction_type = $2 AND trading_strategy=$3", date, transType, tradingStrategy)
	return sumOfSpent, err
}

func (r *Transaction) SaveTransaction(trnsctn *domains.Transaction) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}

	if trnsctn.Id == 0 {
		transactionId := int64(0)
		err := tx.QueryRow("INSERT INTO transaction_table (coin_id, transaction_type, amount, price, total_cost, created_at, client_order_id, api_error, related_transaction_id, profit, percent_profit, commission, trading_strategy, futures_type, stop_loss_price, take_profit_price, fake) values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17) RETURNING id",
			trnsctn.CoinId, trnsctn.TransactionType, trnsctn.Amount, trnsctn.Price, trnsctn.TotalCost, trnsctn.CreatedAt, trnsctn.ClientOrderId, trnsctn.ApiError, trnsctn.RelatedTransactionId, trnsctn.Profit, trnsctn.PercentProfit, trnsctn.Commission, trnsctn.TradingStrategy, trnsctn.FuturesType, trnsctn.StopLossPrice, trnsctn.TakeProfitPrice, trnsctn.IsFake,
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

	return tx.Commit()
}
