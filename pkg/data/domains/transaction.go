package domains

import (
	"cryptoBot/pkg/constants"
	"database/sql"
	"fmt"
	"time"
)

type Transaction struct {
	Id int64

	CoinId int64 `db:"coin_id"`

	TransactionType constants.TransactionType `db:"transaction_type"`

	Amount float64

	Price int64

	/* TotalCost=(amount * price) */
	TotalCost int64 `db:"total_cost"`

	Commission int64

	CreatedAt time.Time `db:"created_at"`

	/* External order id in Binance or Bybit for easy search */
	ClientOrderId sql.NullString `db:"client_order_id"`

	/* api error*/
	ApiError sql.NullString `db:"api_error"`

	/* SELL transaction must contain link to BUY transaction and the opposite */
	RelatedTransactionId sql.NullInt64 `db:"related_transaction_id"`

	/* SELL.TotalCost - BUY.TotalCost - 2 commissions */
	Profit sql.NullInt64

	/* (Profit)/BUY.TotalCost * 100% */
	PercentProfit sql.NullFloat64 `db:"percent_profit"`

	TradingStrategy constants.TradingStrategy `db:"trading_strategy"`

	FuturesType constants.FuturesType `db:"futures_type"`
}

func (t *Transaction) String() string {
	return fmt.Sprintf("Transaction {id: %v, type: %v, coin: %v, amount: %v, price: %v}", t.Id, t.TransactionType, t.CoinId, t.Amount, t.Price)
}
