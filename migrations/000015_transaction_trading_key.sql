-- +migrate Up
ALTER TABLE transaction_table
    ADD trading_key text;