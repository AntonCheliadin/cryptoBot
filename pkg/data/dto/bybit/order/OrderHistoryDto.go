package order

import (
	"cryptoBot/pkg/util"
	"strconv"
	"time"
)

type OrderHistoryDto struct {
	RetCode int         `json:"ret_code"`
	RetMsg  string      `json:"ret_msg"`
	ExtCode interface{} `json:"ext_code"`
	ExtInfo interface{} `json:"ext_info"`
	Result  []struct {
		AccountId           string `json:"accountId"`
		ExchangeId          string `json:"exchangeId"`
		Symbol              string `json:"symbol"`
		SymbolName          string `json:"symbolName"`
		OrderLinkId         string `json:"orderLinkId"`
		OrderId             string `json:"orderId"`
		Price               string `json:"price"`
		OrigQty             string `json:"origQty"`
		ExecutedQty         string `json:"executedQty"`
		CummulativeQuoteQty string `json:"cummulativeQuoteQty"`
		AvgPrice            string `json:"avgPrice"`
		Status              string `json:"status"`
		TimeInForce         string `json:"timeInForce"`
		Type                string `json:"type"`
		Side                string `json:"side"`
		StopPrice           string `json:"stopPrice"`
		IcebergQty          string `json:"icebergQty"`
		Time                string `json:"time"`
		UpdateTime          string `json:"updateTime"`
		IsWorking           bool   `json:"isWorking"`
	} `json:"result"`
}

func (d *OrderHistoryDto) CalculateAvgPrice() int64 {
	return util.GetCentsFromString(d.Result[0].AvgPrice)
}

func (d *OrderHistoryDto) CalculateTotalCost() int64 {
	return int64(float64(d.CalculateAvgPrice()) * d.GetAmount())
}

func (d *OrderHistoryDto) CalculateCommissionInUsd() int64 {
	return int64(float64(d.CalculateTotalCost()) * 0.001) // 0.1% for taker and maker
}

func (d *OrderHistoryDto) GetAmount() float64 {
	amount, _ := strconv.ParseFloat(d.Result[0].OrigQty, 64)
	return amount
}

func (d *OrderHistoryDto) GetCreatedAt() *time.Time {
	return nil
}
