package postgres

import (
	"cryptoBot/pkg/data/domains"
	"fmt"
	"github.com/jmoiron/sqlx"
	"strconv"
	"time"
)

func NewSyntheticKline(db *sqlx.DB) *SyntheticKline {
	return &SyntheticKline{db: db}
}

type SyntheticKline struct {
	db *sqlx.DB
}

func (r *SyntheticKline) FindAllByCoinIdsAndIntervalAndCloseTimeInRange(coinId1 int64, coinId2 int64, interval string, openTime time.Time, closeTime time.Time) ([]domains.IKline, error) {
	var klines []domains.SyntheticKline
	err := r.db.Select(&klines, "SELECT * FROM synthetic_kline WHERE coin_id_1 = $1 AND coin_id_2 = $2 AND duration = $3 AND close_time >= $4 AND close_time <= $5 ORDER BY close_time ASC",
		coinId1, coinId2, interval, openTime, closeTime)

	if err != nil {
		return nil, fmt.Errorf("Error during select domain: %s", err.Error())
	}

	return listRelationsToListRelationsPointersForSyntheticKline(klines), nil
}
func (r *SyntheticKline) FindAllByCoinIdAndIntervalAndCloseTimeLessOrderByOpenTimeWithLimit(coinId1 int64, coinId2 int64, interval string, closeTime time.Time, limit int) ([]domains.IKline, error) {
	intervalInt, _ := strconv.Atoi(interval)
	minCloseTime := closeTime.Add(time.Minute * time.Duration(-intervalInt*(limit+1))) //performance optimization

	var klines []domains.SyntheticKline
	err := r.db.Select(&klines, "SELECT * FROM synthetic_kline WHERE coin_id_1 = $1 AND coin_id_2 = $2 AND duration = $3 AND close_time <= $4 AND close_time >= $5 ORDER BY close_time DESC LIMIT $6",
		coinId1, coinId2, interval, closeTime, minCloseTime, limit)

	if err != nil {
		return nil, fmt.Errorf("Error during select domain: %s", err.Error())
	}

	return listRelationsToListRelationsPointersForSyntheticKline(klines), nil
}

func listRelationsToListRelationsPointersForSyntheticKline(domainList []domains.SyntheticKline) []domains.IKline {
	result := make([]domains.IKline, 0, len(domainList))
	for i := len(domainList) - 1; i >= 0; i-- {
		result = append(result, &domainList[i])
	}
	return result
}
