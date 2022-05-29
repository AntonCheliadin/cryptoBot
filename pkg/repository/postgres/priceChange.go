package postgres

import (
	"cryptoBot/pkg/data/domains"
	"fmt"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
	"strings"
)

func NewPriceChange(db *sqlx.DB) *PriceChange {
	return &PriceChange{db: db}
}

type PriceChange struct {
	db *sqlx.DB
}

func (r *PriceChange) FindByTransactionId(transactionId int64) (*domains.PriceChange, error) {
	var domain domains.PriceChange
	if err := r.db.Get(&domain, "SELECT * FROM price_change WHERE transaction_id=$1", transactionId); err != nil {
		if strings.Contains(err.Error(), "no rows in result set") {
			return nil, nil
		}
		return nil, err
	}
	return &domain, nil
}

func (r *PriceChange) SavePriceChange(domain *domains.PriceChange) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}

	if domain.Id == 0 {
		priceChangeId := int64(0)
		err := tx.QueryRow("INSERT INTO price_change (transaction_id, low_price, high_price, change_percents) values ($1, $2, $3, $4) RETURNING id",
			domain.TransactionId, domain.LowPrice, domain.HighPrice, domain.ChangePercents,
		).Scan(&priceChangeId)
		if err != nil {
			_ = tx.Rollback()
			zap.S().Errorf("Invalid try to save Domain on proxy side: %s. "+
				"Error: %s", domain.String(), err.Error())
			return err
		}
		domain.Id = priceChangeId
		zap.S().Debugf("Domain was saved on proxy side: %s", domain.String())
		return tx.Commit()
	}

	resp, err := tx.Exec("UPDATE price_change SET transaction_id = $2, low_price = $3, high_price = $4 , change_percents = $5 WHERE id = $1",
		domain.Id, domain.TransactionId, domain.LowPrice, domain.HighPrice, domain.ChangePercents)
	if err != nil {
		_ = tx.Rollback()
		zap.S().Errorf("Invalid try to update domain on proxy side: %s. "+
			"Error: %s", domain.String(), err.Error())
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
