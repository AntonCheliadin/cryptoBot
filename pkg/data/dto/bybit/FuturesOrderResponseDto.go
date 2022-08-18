package bybit

import (
	"time"
)

type FuturesOrderResponseDto struct {
	RetCode int    `json:"ret_code"`
	RetMsg  string `json:"ret_msg"`
	ExtCode string `json:"ext_code"`
	ExtInfo string `json:"ext_info"`
	Result  struct {
		UserId        int       `json:"user_id"`
		OrderId       string    `json:"order_id"`
		Symbol        string    `json:"symbol"`
		Side          string    `json:"side"`
		OrderType     string    `json:"order_type"`
		Price         float64   `json:"price"`
		Qty           float64   `json:"qty"`
		TimeInForce   string    `json:"time_in_force"`
		OrderStatus   string    `json:"order_status"`
		LastExecTime  int       `json:"last_exec_time"`
		LastExecPrice int       `json:"last_exec_price"`
		LeavesQty     int       `json:"leaves_qty"`
		CumExecQty    int       `json:"cum_exec_qty"`
		CumExecValue  int       `json:"cum_exec_value"`
		CumExecFee    int       `json:"cum_exec_fee"`
		RejectReason  string    `json:"reject_reason"`
		OrderLinkId   string    `json:"order_link_id"`
		CreatedAt     time.Time `json:"created_at"`
		UpdatedAt     time.Time `json:"updated_at"`
		TakeProfit    float64   `json:"take_profit"`
		StopLoss      float64   `json:"stop_loss"`
		TpTriggerBy   string    `json:"tp_trigger_by"`
		SlTriggerBy   string    `json:"sl_trigger_by"`
	} `json:"result"`
	TimeNow          string `json:"time_now"`
	RateLimitStatus  int    `json:"rate_limit_status"`
	RateLimitResetMs int64  `json:"rate_limit_reset_ms"`
	RateLimit        int    `json:"rate_limit"`
}

func (d *FuturesOrderResponseDto) CalculateAvgPrice() int64 {
	return int64(d.Result.Price)
}

func (d *FuturesOrderResponseDto) CalculateTotalCost() int64 {
	return int64(d.Result.Qty)
}

func (d *FuturesOrderResponseDto) CalculateCommissionInUsd() int64 {
	return int64(float64(d.CalculateTotalCost()) * 0.001) // 0.1% for taker and maker
}

func (d *FuturesOrderResponseDto) GetAmount() float64 {
	return float64(d.CalculateTotalCost()) / float64(d.CalculateAvgPrice())
}
