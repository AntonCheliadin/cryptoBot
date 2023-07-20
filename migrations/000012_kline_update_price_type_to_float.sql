-- +migrate Up
ALTER TABLE kline
    ALTER COLUMN open TYPE decimal;
ALTER TABLE kline
    ALTER COLUMN close TYPE decimal;
ALTER TABLE kline
    ALTER COLUMN low TYPE decimal;
ALTER TABLE kline
    ALTER COLUMN high TYPE decimal;

-- +migrate Up
update kline
set open  = open * 0.01,
    close = close * 0.01,
    low   = low * 0.01,
    high  = high * 0.01
;

-- +migrate Up
ALTER TABLE transaction_table
    ALTER COLUMN price TYPE decimal;
ALTER TABLE transaction_table
    ALTER COLUMN commission TYPE decimal;
ALTER TABLE transaction_table
    ALTER COLUMN total_cost TYPE decimal;
ALTER TABLE transaction_table
    ALTER COLUMN take_profit_price TYPE decimal;
ALTER TABLE transaction_table
    ALTER COLUMN stop_loss_price TYPE decimal;

-- +migrate Up
update transaction_table
set price             = price * 0.01,
    commission        = commission * 0.01,
    total_cost        = total_cost * 0.01,
    take_profit_price = take_profit_price * 0.01,
    stop_loss_price   = stop_loss_price * 0.01
;