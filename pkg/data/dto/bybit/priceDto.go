package bybit

import "github.com/shopspring/decimal"

type PriceDto struct {
	RetCode int         `json:"ret_code"`
	RetMsg  interface{} `json:"ret_msg"`
	Result  struct {
		Symbol string `json:"symbol"`
		Price  string `json:"price"`
	} `json:"result"`
	ExtCode interface{} `json:"ext_code"`
	ExtInfo interface{} `json:"ext_info"`
}

func (d *PriceDto) PriceInCents() (int64, error) {
	price, err := decimal.NewFromString(d.Result.Price)
	if err != nil {
		return 0, err
	}
	return decimal.NewFromInt(100).Mul(price).IntPart(), nil
}
