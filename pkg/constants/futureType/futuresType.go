package futureType

import "github.com/sdcoffey/big"

type FuturesType int8

const (
	LONG FuturesType = iota
	SHORT
)

// GetFuturesSign instead of doing migration
func GetFuturesSign(futuresType FuturesType) int {
	if futuresType == LONG {
		return 1
	} else {
		return -1
	}
}

func GetFuturesSignFloat64(futuresType FuturesType) float64 {
	return float64(GetFuturesSign(futuresType))
}
func GetFuturesSignDecimal(futuresType FuturesType) big.Decimal {
	return big.NewDecimal(GetFuturesSignFloat64(futuresType))
}

func GetString(futuresType FuturesType) string {
	if futuresType == LONG {
		return "LONG"
	} else {
		return "SHORT"
	}
}

func GetTypeByBool(isLong bool) FuturesType {
	if isLong {
		return LONG
	} else {
		return SHORT
	}
}
