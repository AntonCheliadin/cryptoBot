-- +migrate Up
ALTER TABLE transaction_table
    ADD stop_loss_price int;

-- +migrate Up
ALTER TABLE transaction_table
    ADD take_profit_price int;