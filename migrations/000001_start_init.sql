-- +migrate Up
create table if not exists coin
(
    id          SERIAL constraint coin_pkey primary key,

    coin_name   text NOT NULL,
    symbol      text NOT NULL
);

-- +migrate Up
create table if not exists transaction_table
(
    id                          SERIAL
        constraint transaction_table_pkey primary key,
    coin_id                     bigint NOT NULL
        constraint coin_id_fkey references coin,

    transaction_type            int NOT NULL,
    amount                      decimal NOT NULL,
    price                       bigint NOT NULL,
    total_cost                  bigint NOT NULL,
    commission                  bigint NOT NULL,
    created_at                  timestamp NOT NULL,
    client_order_id             text,
    api_error                   text,
    related_transaction_id      bigint,
    profit                      bigint,
    percent_profit              decimal
);


-- +migrate Up
CREATE INDEX coin_symbol_idx ON coin (symbol);

-- +migrate Up
CREATE INDEX transaction_table_coin_date_idx ON transaction_table (coin_id, created_at);


-- +migrate Down
DROP TABLE transaction_table;

-- +migrate Down
DROP TABLE coin;


-- +migrate Down
DROP INDEX coin_symbol_idx;
-- +migrate Down
DROP INDEX transaction_table_coin_date_idx;
