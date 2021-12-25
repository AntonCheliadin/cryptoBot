package postgres

import (
	"github.com/jmoiron/sqlx"
	"strings"
	"tradingBot/pkg/data/domains"
)

func NewCoin(db *sqlx.DB) *Coin {
	return &Coin{db: db}
}

type Coin struct {
	db *sqlx.DB
}

func (r *Coin) FindBySymbol(symbol string) (*domains.Coin, error) {
	var c domains.Coin
	if err := r.db.Get(&c, "SELECT * FROM coin WHERE symbol=$1", symbol); err != nil {
		if strings.Contains(err.Error(), "no rows in result set") {
			return nil, nil
		}
		return nil, err
	}
	return &c, nil
}
