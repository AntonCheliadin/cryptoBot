package domains

import (
	"fmt"
	"time"
)

type Kline struct {
	Id     int64
	CoinId int64 `db:"coin_id"`

	OpenTime  time.Time `db:"open_time"`
	CloseTime time.Time `db:"close_time"`
	Interval  string

	Open  int64
	High  int64
	Low   int64
	Close int64
}

func (d *Kline) String() string {
	return fmt.Sprintf("Kline {id: %v, coin: %v, openTime: %v, interval: %v, open: %v, high: %v, low: %v, close: %v}",
		d.Id, d.CoinId, d.CoinId, d.OpenTime, d.Interval, d.Open, d.High, d.Low, d.Close)
}
