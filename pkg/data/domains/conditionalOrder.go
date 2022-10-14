package domains

import (
	"cryptoBot/pkg/constants"
	"database/sql"
	"fmt"
	"time"
)

type ConditionalOrder struct {
	Id int64

	CoinId int64 `db:"coin_id"`

	TransactionType constants.TransactionType `db:"transaction_type"`

	Amount float64

	StopLossPrice   int64 `db:"stop_loss_price"`
	TakeProfitPrice int64 `db:"take_profit_price"`

	CreatedAt time.Time `db:"created_at"`

	/* External order id in Binance or Bybit for easy search */
	ClientOrderId sql.NullString `db:"client_order_id"`

	/* api error*/
	ApiError sql.NullString `db:"api_error"`

	/* Transaction id of the open order */
	RelatedTransactionId sql.NullInt64 `db:"related_transaction_id"`
}

func (t *ConditionalOrder) String() string {
	return fmt.Sprintf("ConditionalOrder {id: %v, relatedTransactionId: %v,, type: %v, coin: %v, amount: %v, stopLossPrice: %v, takeProfitPrice: %v}", t.Id, t.RelatedTransactionId, t.TransactionType, t.CoinId, t.Amount, t.StopLossPrice, t.TakeProfitPrice)
}
