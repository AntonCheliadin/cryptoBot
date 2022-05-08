-- +migrate Up
ALTER TABLE transaction_table ADD trading_strategy int NOT NULL DEFAULT 0;

-- +migrate Up
ALTER TABLE transaction_table ADD futures_type int;