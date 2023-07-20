package domains

import (
	"fmt"
	"strconv"
	"time"
)

type SyntheticKline struct {
	CoinId1 int64 `db:"coin_id_1"`
	CoinId2 int64 `db:"coin_id_2"`

	OpenTime  time.Time `db:"open_time"`
	CloseTime time.Time `db:"close_time"`
	Interval  string    `db:"duration"`

	Open1  float64 `db:"open_1"`
	Close1 float64 `db:"close_1"`

	Open2  float64 `db:"open_2"`
	Close2 float64 `db:"close_2"`

	SyntheticOpen  float64 `db:"synthetic_open"`
	SyntheticClose float64 `db:"synthetic_close"`
}

func (d *SyntheticKline) String() string {
	return fmt.Sprintf("SyntheticKline {openTime: %v, interval: %v, open: %v, close: %v}",
		d.OpenTime, d.Interval, d.SyntheticOpen, d.SyntheticClose)
}

func (d *SyntheticKline) GetIntervalInMinutes() int64 {
	parsedInt, _ := strconv.ParseInt(d.Interval, 10, 64)

	return parsedInt
}

func (d *SyntheticKline) GetPriceChange() float64 {
	return d.SyntheticClose - d.SyntheticOpen
}
func (d *SyntheticKline) GetOpenTime() time.Time {
	return d.OpenTime
}
func (d *SyntheticKline) GetCloseTime() time.Time {
	return d.CloseTime
}
func (d *SyntheticKline) GetInterval() string {
	return d.Interval
}

// GetOpen Price doesn't matter for synthetic klines, just relation of two prices
func (d *SyntheticKline) GetOpen() float64 {
	return d.SyntheticOpen * 10000 * 10000
}
func (d *SyntheticKline) GetClose() float64 {
	return d.SyntheticClose * 10000 * 10000
}
