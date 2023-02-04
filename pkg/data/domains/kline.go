package domains

import (
	"fmt"
	"strconv"
	"time"
)

type Kline struct {
	Id     int64
	CoinId int64 `db:"coin_id"`

	OpenTime  time.Time `db:"open_time"`
	CloseTime time.Time `db:"close_time"`
	Interval  string    //Data refresh interval. Enum : 1 3 5 15 30 60 120 240 360 720 "D" "M" "W"

	Open  int64
	High  int64
	Low   int64
	Close int64

	Volume float64
}

func (d *Kline) String() string {
	return fmt.Sprintf("Kline {id: %v, coin: %v, openTime: %v, interval: %v, open: %v, high: %v, low: %v, close: %v, volume: %v}",
		d.Id, d.CoinId, d.OpenTime, d.Interval, d.Open, d.High, d.Low, d.Close, d.Volume)
}

func (d *Kline) GetIntervalInMinutes() int64 {
	parsedInt, _ := strconv.ParseInt(d.Interval, 10, 64)

	return parsedInt
}

func (d *Kline) GetPriceChange() int64 {
	return d.Close - d.Open
}
