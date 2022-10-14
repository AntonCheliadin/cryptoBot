package position

import (
	"cryptoBot/pkg/util"
	"time"
)

type GetTradeRecordsDto struct {
	RetCode int    `json:"ret_code"`
	RetMsg  string `json:"ret_msg"`
	ExtCode string `json:"ext_code"`
	ExtInfo string `json:"ext_info"`

	Result struct {
		CurrentPage int              `json:"current_page"`
		Data        []TradeRecordDto `json:"data"`
	} `json:"result"`

	TimeNow          string `json:"time_now"`
	RateLimitStatus  int    `json:"rate_limit_status"`
	RateLimitResetMs int64  `json:"rate_limit_reset_ms"`
	RateLimit        int    `json:"rate_limit"`
}

type TradeRecordDto struct {
	OrderId          string  `json:"order_id"`
	OrderLinkId      string  `json:"order_link_id"`
	Side             string  `json:"side"`
	Symbol           string  `json:"symbol"`
	ExecId           string  `json:"exec_id"`
	Price            float64 `json:"price"`
	OrderPrice       float64 `json:"order_price"`
	OrderQty         float64 `json:"order_qty"`
	OrderType        string  `json:"order_type"`
	FeeRate          float64 `json:"fee_rate"`
	ExecPrice        float64 `json:"exec_price"`
	ExecType         string  `json:"exec_type"`
	ExecQty          float64 `json:"exec_qty"`
	ExecFee          float64 `json:"exec_fee"`
	ExecValue        float64 `json:"exec_value"`
	LeavesQty        float64 `json:"leaves_qty"`
	ClosedSize       float64 `json:"closed_size"`
	LastLiquidityInd string  `json:"last_liquidity_ind"`
	TradeTime        int     `json:"trade_time"`
	TradeTimeMs      int64   `json:"trade_time_ms"`
}

type TradesSummaryDto struct {
	Trades []TradeRecordDto
}

func (dto *TradesSummaryDto) CalculateAvgPrice() int64 {
	return int64(float64(dto.CalculateTotalCost()) / dto.GetAmount())
}

func (dto *TradesSummaryDto) CalculateTotalCost() int64 {
	sumAmount := int64(0)
	for _, trade := range dto.Trades {
		sumAmount += util.GetCents(trade.ExecValue)
	}
	return sumAmount
}

func (dto *TradesSummaryDto) CalculateCommissionInUsd() int64 {
	sumAmount := int64(0)
	for _, trade := range dto.Trades {
		sumAmount += util.GetCents(trade.ExecFee)
	}
	return sumAmount
}

func (dto *TradesSummaryDto) GetAmount() float64 {
	sumAmount := float64(0)
	for _, trade := range dto.Trades {
		sumAmount += trade.ExecQty
	}
	return sumAmount
}

func (dto *TradesSummaryDto) GetCreatedAt() *time.Time {
	timeByMillis := util.GetTimeByMillis(dto.Trades[0].TradeTimeMs)
	return &timeByMillis
}
