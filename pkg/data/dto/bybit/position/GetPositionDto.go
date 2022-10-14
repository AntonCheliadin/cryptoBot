package position

type GetPositionDto struct {
	RetCode          int           `json:"ret_code"`
	RetMsg           string        `json:"ret_msg"`
	ExtCode          string        `json:"ext_code"`
	ExtInfo          string        `json:"ext_info"`
	Result           []PositionDto `json:"result"`
	TimeNow          string        `json:"time_now"`
	RateLimitStatus  int           `json:"rate_limit_status"`
	RateLimitResetMs int64         `json:"rate_limit_reset_ms"`
	RateLimit        int           `json:"rate_limit"`
}

type PositionDto struct {
	UserId              int     `json:"user_id"`
	Symbol              string  `json:"symbol"`
	Side                string  `json:"side"`
	Size                int     `json:"size"`
	PositionValue       float64 `json:"position_value"`
	EntryPrice          float64 `json:"entry_price"`
	LiqPrice            float64 `json:"liq_price"`
	BustPrice           float64 `json:"bust_price"`
	Leverage            int     `json:"leverage"`
	AutoAddMargin       int     `json:"auto_add_margin"`
	IsIsolated          bool    `json:"is_isolated"`
	PositionMargin      float64 `json:"position_margin"`
	OccClosingFee       float64 `json:"occ_closing_fee"`
	RealisedPnl         float64 `json:"realised_pnl"`
	CumRealisedPnl      float64 `json:"cum_realised_pnl"`
	FreeQty             int     `json:"free_qty"`
	TpSlMode            string  `json:"tp_sl_mode"`
	UnrealisedPnl       float64 `json:"unrealised_pnl"`
	DeleverageIndicator int     `json:"deleverage_indicator"`
	RiskId              int     `json:"risk_id"`
	StopLoss            float64 `json:"stop_loss"`
	TakeProfit          float64 `json:"take_profit"`
	TrailingStop        int     `json:"trailing_stop"`
	PositionIdx         int     `json:"position_idx"`
	Mode                string  `json:"mode"`
	TpTriggerBy         int     `json:"tp_trigger_by,omitempty"`
	SlTriggerBy         int     `json:"sl_trigger_by,omitempty"`
}
