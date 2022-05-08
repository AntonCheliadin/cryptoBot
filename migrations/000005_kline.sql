-- +migrate Up
create table if not exists kline
(
    id         SERIAL
        constraint kline_pkey primary key,

    coin_id    bigint    NOT NULL
        constraint kline_coin_fkey references coin,

    open_time  timestamp NOT NULL,
    close_time timestamp NOT NULL,
    interval   text      NOT NULL,

    open       bigint    NOT NULL,
    high       bigint    NOT NULL,
    low        bigint    NOT NULL,
    close      bigint    NOT NULL
);

-- +migrate Up
CREATE INDEX coin_id_idx ON kline (coin_id);
