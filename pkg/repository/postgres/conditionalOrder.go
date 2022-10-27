package postgres

import (
	"cryptoBot/pkg/data/domains"
	"fmt"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
	"strings"
)

func NewConditionalOrder(db *sqlx.DB) *ConditionalOrder {
	return &ConditionalOrder{db: db}
}

type ConditionalOrder struct {
	db *sqlx.DB
}

//language=SQL
func (r *ConditionalOrder) FindByTransaction(transaction *domains.Transaction) (*domains.ConditionalOrder, error) {
	var order domains.ConditionalOrder
	if err := r.db.Get(&transaction, "SELECT * FROM conditional_order WHERE related_transaction_id=$1 limit 1", transaction.Id); err != nil {
		if strings.Contains(err.Error(), "no rows in result set") {
			return nil, nil
		}
		return nil, err
	}
	return &order, nil
}

//language=SQL
func (r *Transaction) SaveConditionalOrder(order *domains.ConditionalOrder) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}

	if order.Id == 0 {
		id := int64(0)
		err := tx.QueryRow("INSERT INTO conditional_order (coin_id, transaction_type, amount, stop_loss_price, take_profit_price, created_at, client_order_id, api_error, related_transaction_id) values ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING id",
			order.CoinId, order.TransactionType, order.Amount, order.StopLossPrice, order.TakeProfitPrice, order.CreatedAt, order.ClientOrderId, order.ApiError, order.RelatedTransactionId,
		).Scan(&id)
		if err != nil {
			_ = tx.Rollback()
			zap.S().Errorf("Invalid try to save Domain on proxy side: %s. "+
				"Error: %s", order.String(), err.Error())
			return err
		}
		order.Id = id
		return tx.Commit()
	}

	resp, err := tx.Exec("UPDATE conditional_order SET coin_id = $2, transaction_type = $3, amount = $4, stop_loss_price = $5, take_profit_price = $6, client_order_id = $7, api_error = $8, related_transaction_id = $9 WHERE id = $1",
		order.Id, order.CoinId, order.TransactionType, order.Amount, order.StopLossPrice, order.TakeProfitPrice, order.ClientOrderId, order.ApiError, order.RelatedTransactionId)
	if err != nil {
		_ = tx.Rollback()
		zap.S().Errorf("Invalid try to update domain on proxy side: %s. "+
			"Error: %s", order.String(), err.Error())
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
