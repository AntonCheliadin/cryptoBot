package transaction

type PairTransactionProfitPercentsDto struct {
	CreatedAt     string  `db:"created_date"`
	ProfitPercent float64 `db:"profit_percent_of_paired_order"`
	ProfitInCents int64   `db:"profit_sum"`
	OrdersSize    int64   `db:"orders_size"`
}
