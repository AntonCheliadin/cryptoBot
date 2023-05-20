package domains

import (
	"cryptoBot/pkg/constants"
	"cryptoBot/pkg/constants/futureType"
	"cryptoBot/pkg/util"
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

	StopLossPrice sql.NullInt64 `db:"stop_loss_price"`

	TakeProfitPrice sql.NullInt64 `db:"take_profit_price"`

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

	FuturesType futureType.FuturesType `db:"futures_type"`

	IsFake bool `db:"fake"`
}

func (t *Transaction) String() string {
	desc := fmt.Sprintf("Transaction {amount: %v, price: %v, cost: %v",
		t.Amount, util.RoundCentsToUsd(t.Price), util.RoundCentsToUsd(t.TotalCost))

	if t.Profit.Valid {
		desc += fmt.Sprintf(", profit: %v", util.RoundCentsToUsd(t.Profit.Int64))
	}
	return desc + "}"
}
