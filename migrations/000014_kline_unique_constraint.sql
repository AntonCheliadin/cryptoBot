-- +migrate Up
ALTER TABLE kline
    ADD CONSTRAINT ucCodes UNIQUE (coin_id, open_time, close_time);