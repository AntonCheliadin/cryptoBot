package domains

type Coin struct {
	Id int64

	Name   string `db:"coin_name"`
	Symbol string
}
