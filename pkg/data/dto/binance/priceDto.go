package binance

import "github.com/shopspring/decimal"

type PriceDto struct {
	Symbol string `json:"symbol"`

	Price string `json:"price"`
}

func (d PriceDto) GetPrice() (float64, error) {
	price, err := decimal.NewFromString(d.Price)
	if err != nil {
		return 0, err
	}
	f, _ := price.Float64()
	return f, nil
}
