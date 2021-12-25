package binance

import (
	"github.com/spf13/viper"
	"strconv"
	"tradingBot/pkg/util"
)

type OrderResponseDto struct {
	Symbol              string `json:"symbol"`
	OrderId             int    `json:"orderId"`
	OrderListId         int    `json:"orderListId"`
	ClientOrderId       string `json:"clientOrderId"`
	TransactTime        int64  `json:"transactTime"`
	Price               string `json:"price"`
	OrigQty             string `json:"origQty"`
	ExecutedQty         string `json:"executedQty"`
	CummulativeQuoteQty string `json:"cummulativeQuoteQty"` // totalCost in USD
	Status              string `json:"status"`
	TimeInForce         string `json:"timeInForce"`
	Type                string `json:"type"`
	Side                string `json:"side"`
	Fills               []fill `json:"fills"`
}

type fill struct {
	Price           string `json:"price"`
	Qty             string `json:"qty"`
	Commission      string `json:"commission"`
	CommissionAsset string `json:"commissionAsset"`
}

func (f fill) getPrice() float64 {
	money, _ := strconv.ParseFloat(f.Price, 64)
	return money
}

func (f fill) getCommission() float64 {
	money, _ := strconv.ParseFloat(f.Commission, 64)
	return money
}

func (d OrderResponseDto) CalculateAvgPrice() int64 {
	price := float64(0)

	for _, fill := range d.Fills {
		price += fill.getPrice()
	}

	return util.GetCents(price / float64(len(d.Fills)))
}

func (d OrderResponseDto) CalculateTotalCost() int64 {
	return util.GetCentsFromString(d.CummulativeQuoteQty)
}

func (d OrderResponseDto) CalculateCommissionInUsd() int64 {
	totalCommission := float64(0)

	for _, fill := range d.Fills {
		totalCommission += fill.getCommission() * viper.GetFloat64("api.binance.commission.bnbCost")
	}

	return util.GetCents(totalCommission)
}

func (d OrderResponseDto) GetAmount() float64 {
	amount, _ := strconv.ParseFloat(d.ExecutedQty, 64)
	return amount
}
