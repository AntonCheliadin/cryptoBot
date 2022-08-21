package bybit

import (
	"cryptoBot/pkg/util"
	"time"
)

type ActiveOrdersResponseDto struct {
	RetCode int    `json:"ret_code"`
	RetMsg  string `json:"ret_msg"`
	ExtCode string `json:"ext_code"`
	ExtInfo string `json:"ext_info"`
	Result  struct {
		CurrentPage int              `json:"current_page"`
		Data        []ActiveOrderDto `json:"data"`
	} `json:"result"`
	TimeNow          string `json:"time_now"`
	RateLimitStatus  int    `json:"rate_limit_status"`
	RateLimitResetMs int64  `json:"rate_limit_reset_ms"`
	RateLimit        int    `json:"rate_limit"`
}

type ActiveOrderDto struct {
	OrderId        string    `json:"order_id"`
	UserId         int       `json:"user_id"`
	Symbol         string    `json:"symbol"`
	Side           string    `json:"side"`
	OrderType      string    `json:"order_type"`
	Price          float64   `json:"price"`
	Qty            float64   `json:"qty"`
	TimeInForce    string    `json:"time_in_force"`
	OrderStatus    string    `json:"order_status"`
	LastExecPrice  float64   `json:"last_exec_price"`
	CumExecQty     float64   `json:"cum_exec_qty"`
	CumExecValue   float64   `json:"cum_exec_value"`
	CumExecFee     float64   `json:"cum_exec_fee"`
	ReduceOnly     bool      `json:"reduce_only"`
	CloseOnTrigger bool      `json:"close_on_trigger"`
	OrderLinkId    string    `json:"order_link_id"`
	CreatedTime    time.Time `json:"created_time"`
	UpdatedTime    time.Time `json:"updated_time"`
	TakeProfit     float64   `json:"take_profit"`
	StopLoss       float64   `json:"stop_loss"`
	TpTriggerBy    string    `json:"tp_trigger_by"`
	SlTriggerBy    string    `json:"sl_trigger_by"`
}

func (d *ActiveOrderDto) CalculateAvgPrice() int64 {
	return util.GetCents(d.LastExecPrice)
}

func (d *ActiveOrderDto) CalculateTotalCost() int64 {
	return util.GetCents(d.CumExecValue)
}

func (d *ActiveOrderDto) CalculateCommissionInUsd() int64 {
	return util.GetCents(d.CumExecFee)
}

func (d *ActiveOrderDto) GetAmount() float64 {
	return d.CumExecQty
}
