package wallet

type GetWalletBalanceDto struct {
	RetCode int    `json:"ret_code"`
	RetMsg  string `json:"ret_msg"`
	ExtCode string `json:"ext_code"`
	ExtInfo string `json:"ext_info"`
	Result  struct {
		USDT struct {
			Equity           float64 `json:"equity"`
			AvailableBalance float64 `json:"available_balance"`
			UsedMargin       float64 `json:"used_margin"`
			OrderMargin      float64 `json:"order_margin"`
			PositionMargin   float64 `json:"position_margin"`
			OccClosingFee    float64 `json:"occ_closing_fee"`
			OccFundingFee    float64 `json:"occ_funding_fee"`
			WalletBalance    float64 `json:"wallet_balance"`
			RealisedPnl      float64 `json:"realised_pnl"`
			UnrealisedPnl    float64 `json:"unrealised_pnl"`
			CumRealisedPnl   float64 `json:"cum_realised_pnl"`
			GivenCash        float64 `json:"given_cash"`
			ServiceCash      float64 `json:"service_cash"`
		} `json:"USDT"`
	} `json:"result"`
	TimeNow          string `json:"time_now"`
	RateLimitStatus  int    `json:"rate_limit_status"`
	RateLimitResetMs int64  `json:"rate_limit_reset_ms"`
	RateLimit        int    `json:"rate_limit"`
}

func (dto *GetWalletBalanceDto) GetAvailableBalanceInCents() float64 {
	return dto.Result.USDT.AvailableBalance
}
