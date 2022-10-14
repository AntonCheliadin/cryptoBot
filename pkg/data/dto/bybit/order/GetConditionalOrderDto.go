package order

import "time"

type GetConditionalOrderDto struct {
	RetCode int    `json:"ret_code"`
	RetMsg  string `json:"ret_msg"`
	ExtCode string `json:"ext_code"`
	Result  struct {
		CurrentPage int `json:"current_page"`
		LastPage    int `json:"last_page"`
		Data        []struct {
			StopOrderId    string    `json:"stop_order_id"`
			UserId         int       `json:"user_id"`
			Symbol         string    `json:"symbol"`
			Side           string    `json:"side"`
			OrderType      string    `json:"order_type"`
			Price          float64   `json:"price"`
			Qty            float64   `json:"qty"`
			TimeInForce    string    `json:"time_in_force"`
			OrderStatus    string    `json:"order_status"`
			TriggerPrice   float64   `json:"trigger_price"`
			OrderLinkId    string    `json:"order_link_id"`
			CreatedTime    time.Time `json:"created_time"`
			UpdatedTime    time.Time `json:"updated_time"`
			TakeProfit     float64   `json:"take_profit"`
			StopLoss       float64   `json:"stop_loss"`
			TpTriggerBy    string    `json:"tp_trigger_by"`
			SlTriggerBy    string    `json:"sl_trigger_by"`
			BasePrice      string    `json:"base_price"`
			TriggerBy      string    `json:"trigger_by"`
			ReduceOnly     bool      `json:"reduce_only,omitempty"`
			CloseOnTrigger bool      `json:"close_on_trigger,omitempty"`
		} `json:"data"`
	} `json:"result"`
	ExtInfo          interface{} `json:"ext_info"`
	TimeNow          string      `json:"time_now"`
	RateLimitStatus  int         `json:"rate_limit_status"`
	RateLimitResetMs int64       `json:"rate_limit_reset_ms"`
	RateLimit        int         `json:"rate_limit"`
}
