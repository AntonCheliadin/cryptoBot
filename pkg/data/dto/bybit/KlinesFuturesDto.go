package bybit

import (
	"cryptoBot/pkg/api"
	"cryptoBot/pkg/util"
	"fmt"
	"strconv"
	"time"
)

type KlinesFuturesDto struct {
	RetCode int    `json:"retCode"`
	RetMsg  string `json:"retMsg"`
	Result  struct {
		Symbol   string     `json:"symbol"`
		Category string     `json:"category"`
		List     [][]string `json:"list"`
	} `json:"result"`

	Time int64 `json:"time"`

	Interval int
}

func (d *KlinesFuturesDto) String() string {
	return fmt.Sprintf("KlinesFuturesDto {RetCode: %v, RetMsg: %v, Symbol: %v}",
		d.RetCode, d.RetMsg, d.Result.Symbol)
}

// GetKlines
//Sort in reverse by start
//The default collation within the array is start, open, high, low, close, volume, turnover
func (dto *KlinesFuturesDto) GetKlines() []api.KlineDto {
	klines := make([]api.KlineDto, len(dto.Result.List), len(dto.Result.List))
	for i, kline := range dto.Result.List {

		startMillis, _ := strconv.Atoi(kline[0])
		open, _ := strconv.ParseFloat(kline[1], 64)
		high, _ := strconv.ParseFloat(kline[2], 64)
		low, _ := strconv.ParseFloat(kline[3], 64)
		closee, _ := strconv.ParseFloat(kline[4], 64)
		volume, _ := strconv.ParseFloat(kline[5], 64)
		turnover, _ := strconv.ParseFloat(kline[6], 64)

		klines[i] = &KlineFuturesDto{
			StartAt:  util.GetTimeByMillis(int64(startMillis)),
			Open:     open,
			High:     high,
			Low:      low,
			Close:    closee,
			Volume:   volume,
			Turnover: turnover,
			Interval: dto.Interval,
		}
	}

	return klines
}

type KlineFuturesDto struct {
	StartAt  time.Time
	Open     float64 `json:"open"`
	High     float64 `json:"high"`
	Low      float64 `json:"low"`
	Close    float64 `json:"close"`
	Volume   float64 `json:"volume"`
	Turnover float64 `json:"turnover"`

	Interval int
}

func (dto *KlineFuturesDto) GetSymbol() string {
	panic("Unexpected GetSymbol in KlineFuturesDto")
}

func (dto *KlineFuturesDto) GetInterval() string {
	panic("Unexpected GetInterval in KlineFuturesDto")
}

func (dto *KlineFuturesDto) GetStartAt() time.Time {
	return dto.StartAt
}

func (dto *KlineFuturesDto) GetCloseAt() time.Time {
	return dto.GetStartAt().Add(time.Minute * time.Duration(dto.Interval))
}

func (dto *KlineFuturesDto) GetOpen() float64 {
	return dto.Open
}

func (dto *KlineFuturesDto) GetHigh() float64 {
	return dto.High
}

func (dto *KlineFuturesDto) GetLow() float64 {
	return dto.Low
}
func (dto *KlineFuturesDto) GetClose() float64 {
	return dto.Close
}
