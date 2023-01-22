package postgres

import (
	"cryptoBot/pkg/data/domains"
	"cryptoBot/pkg/util"
	"fmt"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
	"strings"
	"time"
)

func NewKline(db *sqlx.DB) *Kline {
	return &Kline{db: db}
}

type Kline struct {
	db *sqlx.DB
}

func (r *Kline) find(query string, arguments ...interface{}) (*domains.Kline, error) {
	var domain domains.Kline
	if err := r.db.Get(&domain, query, arguments); err != nil {
		if strings.Contains(err.Error(), "no rows in result set") {
			return nil, nil
		}
		return nil, err
	}
	return &domain, nil
}

//language=SQL
func (r *Kline) FindOpenedAtMoment(coinId int64, momentTime time.Time, interval string) (*domains.Kline, error) {
	var domain domains.Kline
	if err := r.db.Get(&domain, "SELECT * FROM kline WHERE coin_id = $1 AND interval = $2 AND open_time = $3", coinId, interval, util.RoundToMinutes(momentTime)); err != nil {
		if strings.Contains(err.Error(), "no rows in result set") {
			return nil, nil
		}
		return nil, err
	}
	return &domain, nil
}

//language=SQL
func (r *Kline) FindClosedAtMoment(coinId int64, momentTime time.Time, interval string) (*domains.Kline, error) {
	var domain domains.Kline
	if err := r.db.Get(&domain, "SELECT * FROM kline WHERE coin_id = $1 AND interval = $2 AND close_time = $3", coinId, interval, util.RoundToMinutesWithInterval(momentTime, interval)); err != nil {
		if strings.Contains(err.Error(), "no rows in result set") {
			return nil, nil
		}
		return nil, err
	}
	return &domain, nil
}

//language=SQL
func (r *Kline) FindLast(coinId int64, interval string) (*domains.Kline, error) {
	var domain domains.Kline
	if err := r.db.Get(&domain, "SELECT * FROM kline WHERE coin_id = $1 AND interval = $2 ORDER BY open_time DESC LIMIT 1", coinId, interval); err != nil {
		if strings.Contains(err.Error(), "no rows in result set") {
			return nil, nil
		}
		return nil, err
	}
	return &domain, nil
}

func (r *Kline) FindAllByCoinIdAndIntervalAndCloseTimeLessOrderByOpenTimeWithLimit(
	coinId int64, interval string, closeTime time.Time, limit int64) ([]*domains.Kline, error) {
	var klines []domains.Kline
	err := r.db.Select(&klines, "SELECT * FROM kline WHERE coin_id = $1 AND interval = $2 AND close_time <= $3 ORDER BY open_time DESC LIMIT $4",
		coinId, interval, closeTime, limit)

	if err != nil {
		return nil, fmt.Errorf("Error during select domain: %s", err.Error())
	}

	return listRelationsToListRelationsPointers(klines), nil
}
func (r *Kline) FindAllByCoinIdAndIntervalAndCloseTimeInRange(coinId int64, interval string, openTime time.Time, closeTime time.Time) ([]*domains.Kline, error) {
	var klines []domains.Kline
	err := r.db.Select(&klines, "SELECT * FROM kline WHERE coin_id = $1 AND interval = $2 AND close_time >= $3 AND close_time <= $4 ORDER BY open_time ASC",
		coinId, interval, openTime, closeTime)

	if err != nil {
		return nil, fmt.Errorf("Error during select domain: %s", err.Error())
	}

	return listRelationsToListRelationsPointers(klines), nil
}

/* Sort ASC */
func listRelationsToListRelationsPointers(domainList []domains.Kline) []*domains.Kline {
	result := make([]*domains.Kline, 0, len(domainList))
	for i := len(domainList) - 1; i >= 0; i-- {
		result = append(result, &domainList[i])
	}
	return result
}

//language=SQL
func (r *Kline) SaveKline(domain *domains.Kline) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}

	if domain.Id == 0 {
		id := int64(0)
		err := tx.QueryRow("INSERT INTO kline (coin_id, open_time, close_time, interval, open, high, low, close, volume) values ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id",
			domain.CoinId, domain.OpenTime, domain.CloseTime, domain.Interval, domain.Open, domain.High, domain.Low, domain.Close, domain.Volume,
		).Scan(&id)
		if err != nil {
			_ = tx.Rollback()
			zap.S().Errorf("Invalid try to save Domain on proxy side: %s. "+
				"Error: %s", domain.String(), err.Error())
			return err
		}
		domain.Id = id
		return tx.Commit()
	}

	resp, err := tx.Exec("UPDATE kline SET close_time = $2, high = $3, low = $4, close = $5, volume = $5 WHERE id = $1",
		domain.Id, domain.CloseTime, domain.High, domain.Low, domain.Close, domain.Volume)
	if err != nil {
		_ = tx.Rollback()
		zap.S().Errorf("Invalid try to update domain on proxy side: %s. "+
			"Error: %s", domain.String(), err.Error())
		return err
	}

	if count, err := resp.RowsAffected(); err != nil {
		_ = tx.Rollback()
		return err
	} else if count != 1 {
		_ = tx.Rollback()
		return fmt.Errorf("Unexpected updated rows count: %d", count)
	}

	return tx.Commit()
}
