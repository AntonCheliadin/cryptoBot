-- +migrate Up
CREATE MATERIALIZED VIEW synthetic_kline AS
select k1.coin_id    coin_id_1,
       k2.coin_id    coin_id_2,
       k1.interval   duration,
       k1.open_time  open_time,
       k1.close_time close_time,
       k1.open open_1,
       k1.close close_1,
       k2.open open_2,
       k2.close close_2,
       (k1.open::float) / (k2.open::float) * 10000 synthetic_open,
       (k1.close::float) / (k2.close::float) * 10000 synthetic_close
from
    kline k1,
    kline k2
where k1.coin_id != k2.coin_id
  and k1.open_time = k2.open_time
  and k1.close_time = k2.close_time
;


-- +migrate Up
CREATE INDEX synthetic_kline_idx ON synthetic_kline (coin_id_1, coin_id_2, duration, close_time);

-- +migrate Up
REFRESH MATERIALIZED VIEW synthetic_kline;
