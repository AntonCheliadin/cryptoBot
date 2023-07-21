package bybit

import "github.com/shopspring/decimal"

type TickerInfoDto struct {
	RetCode int    `json:"retCode"`
	RetMsg  string `json:"retMsg"`
	Result  struct {
		Category string `json:"category"`
		List     []struct {
			Symbol                 string `json:"symbol"`
			BidPrice               string `json:"bidPrice"`
			AskPrice               string `json:"askPrice"`
			LastPrice              string `json:"lastPrice"`
			LastTickDirection      string `json:"lastTickDirection"`
			PrevPrice24H           string `json:"prevPrice24h"`
			Price24HPcnt           string `json:"price24hPcnt"`
			HighPrice24H           string `json:"highPrice24h"`
			LowPrice24H            string `json:"lowPrice24h"`
			PrevPrice1H            string `json:"prevPrice1h"`
			MarkPrice              string `json:"markPrice"`
			IndexPrice             string `json:"indexPrice"`
			OpenInterest           string `json:"openInterest"`
			Turnover24H            string `json:"turnover24h"`
			Volume24H              string `json:"volume24h"`
			FundingRate            string `json:"fundingRate"`
			NextFundingTime        string `json:"nextFundingTime"`
			PredictedDeliveryPrice string `json:"predictedDeliveryPrice"`
			BasisRate              string `json:"basisRate"`
			DeliveryFeeRate        string `json:"deliveryFeeRate"`
			DeliveryTime           string `json:"deliveryTime"`
			OpenInterestValue      string `json:"openInterestValue"`
		} `json:"list"`
	} `json:"result"`
	RetExtInfo struct {
	} `json:"retExtInfo"`
	Time int64 `json:"time"`
}

func (d *TickerInfoDto) Price() (float64, error) {
	price, err := decimal.NewFromString(d.Result.List[0].MarkPrice)
	if err != nil {
		return 0, err
	}
	return price.InexactFloat64(), nil
}
