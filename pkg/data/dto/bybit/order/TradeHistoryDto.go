package order

import (
	"cryptoBot/pkg/util"
	"fmt"
	"math"
	"strconv"
	"time"
)

type TradeHistoryDto struct {
	RetCode int         `json:"ret_code"`
	RetMsg  string      `json:"ret_msg"`
	ExtCode interface{} `json:"ext_code"`
	ExtInfo interface{} `json:"ext_info"`
	Result  []struct {
		Id              string `json:"id"`
		Symbol          string `json:"symbol"`
		SymbolName      string `json:"symbolName"`
		OrderId         string `json:"orderId"`
		TicketId        string `json:"ticketId"`
		MatchOrderId    string `json:"matchOrderId"`
		Price           string `json:"price"`
		Qty             string `json:"qty"`
		Commission      string `json:"commission"`
		CommissionAsset string `json:"commissionAsset"`
		Time            string `json:"time"`
		IsBuyer         bool   `json:"isBuyer"`
		IsMaker         bool   `json:"isMaker"`
		Fee             struct {
			FeeTokenId   string `json:"feeTokenId"`
			FeeTokenName string `json:"feeTokenName"`
			Fee          string `json:"fee"`
		} `json:"fee"`
		FeeTokenId    string `json:"feeTokenId"`
		FeeAmount     string `json:"feeAmount"`
		MakerRebate   string `json:"makerRebate"`
		ExecutionTime string `json:"executionTime"`
	} `json:"result"`
}

func (d *TradeHistoryDto) CalculateAvgPrice() int64 {
	return int64(float64(d.CalculateTotalCost()) / d.GetAmount())
}

func (d *TradeHistoryDto) CalculateTotalCost() int64 {
	sumCost := int64(0)
	for _, trade := range d.Result {
		amount, _ := strconv.ParseFloat(trade.Qty, 64)
		price, _ := strconv.ParseFloat(trade.Price, 64)

		sumCost += int64(amount * price * 100)
	}
	return sumCost
}

func (d *TradeHistoryDto) CalculateCommissionInUsd() int64 {
	sum := float64(0)
	for _, trade := range d.Result {
		commission, _ := strconv.ParseFloat(trade.Commission, 64)

		sum += commission
	}
	return int64(sum * 100)
}

func (d *TradeHistoryDto) GetAmount() float64 {
	sumAmount := float64(0)
	for _, trade := range d.Result {
		amount, _ := strconv.ParseFloat(trade.Qty, 64)

		sumAmount += amount
	}
	return math.Round(sumAmount*10000000) / 10000000
}

func (d *TradeHistoryDto) GetCreatedAt() *time.Time {
	millis, _ := strconv.ParseInt(d.Result[0].ExecutionTime, 10, 64)
	timeByMillis := util.GetTimeByMillis(millis)
	return &timeByMillis
}

func (d *TradeHistoryDto) String() string {
	return fmt.Sprintf("TradeHistoryDto {CalculateAvgPrice: %v, CalculateTotalCost: %v, CalculateCommissionInUsd: %v, GetAmount: %v, GetCreatedAt: %v}",
		d.CalculateAvgPrice(), d.CalculateTotalCost(), d.CalculateCommissionInUsd(), d.GetAmount(), d.GetCreatedAt())
}
