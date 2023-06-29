package transaction

import "time"

type TransactionProfitPercentsDto struct {
	CreatedAt     time.Time `db:"created_at"`
	ProfitPercent float64   `db:"profit_percent"`
}
