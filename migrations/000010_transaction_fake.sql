-- +migrate Up
ALTER TABLE transaction_table
    ADD fake boolean NOT NULL DEFAULT false;