package postgres

import (
	"cryptoBot/pkg/data/domains"
	"fmt"
	"github.com/jmoiron/sqlx"
	"time"
)

func NewSyntheticKline(db *sqlx.DB) *SyntheticKline {
	return &SyntheticKline{db: db}
}

type SyntheticKline struct {
	db *sqlx.DB
}

func (r *SyntheticKline) FindAllSyntheticKlinesByCoinIdsAndIntervalAndCloseTimeInRange(coinId1 int64, coinId2 int64, interval string, openTime time.Time, closeTime time.Time) ([]*domains.SyntheticKline, error) {
	var klines []domains.SyntheticKline
	err := r.db.Select(&klines, "SELECT * FROM synthetic_kline WHERE coin_id_1 = $1 AND coin_id_2 = $2 AND duration = $3 AND close_time >= $4 AND close_time <= $5 ORDER BY open_time ASC",
		coinId1, coinId2, interval, openTime, closeTime)

	if err != nil {
		return nil, fmt.Errorf("Error during select domain: %s", err.Error())
	}

	return listRelationsToListRelationsPointersForSyntheticKline(klines), nil
}

func listRelationsToListRelationsPointersForSyntheticKline(domainList []domains.SyntheticKline) []*domains.SyntheticKline {
	result := make([]*domains.SyntheticKline, 0, len(domainList))
	for i := len(domainList) - 1; i >= 0; i-- {
		result = append(result, &domainList[i])
	}
	return result
}
