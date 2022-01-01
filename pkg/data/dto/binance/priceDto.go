package binance

import "github.com/shopspring/decimal"

type PriceDto struct {
	Symbol string `json:"symbol"`

	Price string `json:"price"`
}

func (d *PriceDto) PriceInCents() (int64, error) {
	price, err := decimal.NewFromString(d.Price)
	if err != nil {
		return 0, err
	}
	return decimal.NewFromInt(100).Mul(price).IntPart(), nil
}
