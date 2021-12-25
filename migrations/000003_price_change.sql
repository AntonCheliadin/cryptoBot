-- +migrate Up
create table if not exists price_change
(
    id             SERIAL
        constraint price_change_pkey primary key,

    transaction_id bigint NOT NULL
        constraint price_change_transaction_fkey references transaction_table,
    low_price      bigint NOT NULL,
    high_price     bigint NOT NULL
);

-- +migrate Up
CREATE INDEX price_change_transaction_id_idx ON price_change (transaction_id);
