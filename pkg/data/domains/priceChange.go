package domains

import (
	"fmt"
	"tradingBot/pkg/util"
)

type PriceChange struct {
	Id             int64
	TransactionId  int64   `db:"transaction_id"`
	LowPrice       int64   `db:"low_price"`
	HighPrice      int64   `db:"high_price"`
	ChangePercents float64 `db:"change_percents"`
}

func (d *PriceChange) SetLow(low int64) {
	d.LowPrice = low
	d.RecalculatePercent()
}

func (d *PriceChange) SetHigh(high int64) {
	d.HighPrice = high
	d.RecalculatePercent()
}

func (d *PriceChange) RecalculatePercent() {
	d.ChangePercents = util.CalculatePercentsAbs(d.LowPrice, d.HighPrice)
}

func (d *PriceChange) String() string {
	return fmt.Sprintf("PriceChange {id: %v, TransactionId: %v, LowPrice: %v, HighPrice: %v, ChangePercents: %v}", d.Id, d.TransactionId, d.LowPrice, d.HighPrice, d.ChangePercents)
}
