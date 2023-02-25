package constants

type TradingStrategy int8

const (
	HOLDER TradingStrategy = iota
	MOVING_AVARAGE
	MOVING_AVARAGE_RESISTANCE
	TREND_METER
	SESSION_SCALPER
	SMA_VOLUME_SCALPER
)
