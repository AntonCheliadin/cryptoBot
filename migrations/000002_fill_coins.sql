-- +migrate Up
insert into coin (coin_name, symbol)
values ('Bitcoin', 'BTCUSDT');
insert into coin (coin_name, symbol)
values ('Binance coin', 'BNBUSDT');