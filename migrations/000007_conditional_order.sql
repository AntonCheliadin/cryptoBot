-- +migrate Up
create table if not exists conditional_order
(
    id                     SERIAL
        constraint conditional_order_pkey primary key,
    coin_id                bigint    NOT NULL
        constraint coin_id_fkey references coin,

    transaction_type       int       NOT NULL,
    amount                 decimal   NOT NULL,
    stop_loss_price        bigint    NOT NULL,
    take_profit_price      bigint    NOT NULL,
    created_at             timestamp NOT NULL,
    client_order_id        text,
    api_error              text,
    related_transaction_id bigint
);

-- +migrate Up
CREATE INDEX co_coin_id_idx ON conditional_order (coin_id);

-- +migrate Up
CREATE INDEX co_related_transaction_id_idx ON conditional_order (related_transaction_id);
