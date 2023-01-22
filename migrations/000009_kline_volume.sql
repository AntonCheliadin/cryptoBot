-- +migrate Up
ALTER TABLE kline
    ADD volume decimal NOT NULL DEFAULT 0;