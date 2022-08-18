package bybit

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"cryptoBot/pkg/api"
	"cryptoBot/pkg/constants"
	"cryptoBot/pkg/data/domains"
	"cryptoBot/pkg/data/dto/bybit"
	"cryptoBot/pkg/util"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

func NewBybitApi() api.ExchangeApi {
	return &BybitApi{}
}

type BybitApi struct {
}

func (bybitApi *BybitApi) GetKlines(coin *domains.Coin, interval string, limit int, fromTime time.Time) (api.KlinesDto, error) {
	resp, err := http.Get("https://api.bytick.com/public/linear/kline?" +
		"symbol=" + coin.Symbol +
		"&interval=" + interval +
		"&limit=" + strconv.Itoa(limit) +
		"&from=" + strconv.Itoa(util.GetSecondsByTime(fromTime)))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var dto bybit.KlinesDto
	if err := json.NewDecoder(resp.Body).Decode(&dto); err != nil {
		return nil, err
	}

	return &dto, nil
}

func (api *BybitApi) GetCurrentCoinPrice(coin *domains.Coin) (int64, error) {
	resp, err := http.Get("https://api.bytick.com/spot/quote/v1/ticker/price?symbol=" + coin.Symbol)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	var priceDto bybit.PriceDto
	if err := json.NewDecoder(resp.Body).Decode(&priceDto); err != nil {
		return 0, err
	}

	return priceDto.PriceInCents()
}

func (api *BybitApi) BuyCoinByMarket(coin *domains.Coin, amount float64, price int64) (api.OrderResponseDto, error) {
	queryParams := api.buildParams(coin, amount, "Buy")
	return api.orderCoinByMarket(queryParams)
}

func (api *BybitApi) SellCoinByMarket(coin *domains.Coin, amount float64, price int64) (api.OrderResponseDto, error) {
	queryParams := api.buildParams(coin, amount, "Sell")
	return api.orderCoinByMarket(queryParams)
}

func (api *BybitApi) orderCoinByMarket(queryParams string) (api.OrderResponseDto, error) {
	body, err := api.postSignedApiRequest("/spot/v1/order?", map[string]interface{}{} /*queryParams*/)
	if err != nil {
		return nil, err
	}

	dto := bybit.OrderResponseDto{}
	errUnmarshal := json.Unmarshal(body, &dto)
	if errUnmarshal != nil {
		zap.S().Error("Unmarshal error", errUnmarshal.Error())
		return nil, errUnmarshal
	}

	time.Sleep(30 * time.Second)

	return api.getOrderDetails(dto)
}

func (api *BybitApi) postSignedApiRequest(uri string, queryParams map[string]interface{}) ([]byte, error) {
	queryParams["sign"] = api.getSignature(queryParams, os.Getenv("BYBIT_CryptoBotFutures_API_SECRET"))
	jsonString, _ := json.Marshal(queryParams)

	urlRequest := "https://api.bytick.com" + uri

	zap.S().Infof("postSignedApiRequest = %s  json= %v", urlRequest, string(jsonString))

	method := http.MethodPost
	client := &http.Client{}
	req, err := http.NewRequest(method, urlRequest, bytes.NewBuffer(jsonString))

	if err != nil {
		zap.S().Errorf("API error: %s", err)
		return nil, err
	}
	req.Header.Add("Content-Type", "application/json")

	res, err := client.Do(req)
	if err != nil {
		zap.S().Errorf("API error: %s", err)
		return nil, err
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		zap.S().Errorf("API error: %s", err)
		return nil, err
	}
	zap.S().Infof("API response: %s", string(body))
	return body, nil
}

func (api *BybitApi) getOrderDetails(orderResponseDto bybit.OrderResponseDto) (api.OrderResponseDto, error) {
	//queryParams := "api_key=" + os.Getenv("BYBIT_CryptoBotSubAcc_API_KEY") +
	//	"&orderId=" + orderResponseDto.Result.OrderId +
	//	"&timestamp=" + util.MakeTimestamp()

	body, err := api.postSignedApiRequest("/spot/v1/history-orders?", map[string]interface{}{})
	if err != nil {
		return nil, err
	}

	dto := bybit.OrderHistoryDto{}
	errUnmarshal := json.Unmarshal(body, &dto)
	if errUnmarshal != nil {
		zap.S().Error("Unmarshal error", errUnmarshal.Error())
		return nil, errUnmarshal
	}

	return &dto, nil
}

func (api *BybitApi) buildParams(coin *domains.Coin, amount float64, side string) string {
	return "api_key=" + os.Getenv("BYBIT_CryptoBotSubAcc_API_KEY") +
		"&qty=" + api.buildQty(amount, side) +
		"&side=" + side +
		"&symbol=" + coin.Symbol +
		"&timestamp=" + util.MakeTimestamp() +
		"&type=MARKET"
}

/**
Order quantity
for market orders: when side is Buy, this is in the quote currency.
Otherwise, qty is in the base currency.
For example, on BTCUSDT a Buy order is in USDT, otherwise it's in BTC. For limit orders, the qty is always in the base currency.
*/
func (api *BybitApi) buildQty(amount float64, side string) string {
	if side == "Buy" {
		return viper.GetString("trading.defaultCost")
	} else {
		return strings.TrimRight(fmt.Sprintf("%f", amount), "0")
	}
}

func (api *BybitApi) sign(data string) string {
	secret := os.Getenv("BYBIT_CryptoBotFutures_API_SECRET")

	// Create a new HMAC by defining the hash type and the key (as byte array)
	h := hmac.New(sha256.New, []byte(secret))

	// Write Data to it
	h.Write([]byte(data))

	// Get result and encode as hexadecimal string
	sha := hex.EncodeToString(h.Sum(nil))

	return sha
}

func (api *BybitApi) getSignature(params map[string]interface{}, key string) string {
	keys := make([]string, len(params))
	i := 0
	_val := ""
	for k, _ := range params {
		keys[i] = k
		i++
	}
	sort.Strings(keys)
	for _, k := range keys {
		_val += k + "=" + fmt.Sprintf("%v", params[k]) + "&"
	}
	_val = _val[0 : len(_val)-1]
	h := hmac.New(sha256.New, []byte(key))
	io.WriteString(h, _val)
	return fmt.Sprintf("%x", h.Sum(nil))
}

func (api *BybitApi) SetFuturesLeverage(coin *domains.Coin, leverage int) error {
	_, err := api.postSignedApiRequest("/private/linear/position/set-leverage",
		map[string]interface{}{
			"api_key":       os.Getenv("BYBIT_CryptoBotFutures_API_KEY"),
			"buy_leverage":  strconv.Itoa(leverage),
			"sell_leverage": strconv.Itoa(leverage),
			"symbol":        coin.Symbol,
			"timestamp":     util.MakeTimestamp(),
		},
	)

	return err
}

func (api *BybitApi) OpenFuturesOrder(coin *domains.Coin, amount float64, price int64, futuresType constants.FuturesType) (api.OrderResponseDto, error) {
	queryParams := api.buildOpenFuturesParams(coin, amount, price, futuresType)
	return api.futuresOrderByMarket(queryParams)
}

func (api *BybitApi) CloseFuturesOrder(coin *domains.Coin, openedTransaction *domains.Transaction, price int64) (api.OrderResponseDto, error) {
	queryParams := api.buildCloseFuturesParams(coin, openedTransaction, price)

	return api.futuresOrderByMarket(queryParams)
}

func (api *BybitApi) buildOpenFuturesParams(coin *domains.Coin, amount float64, priceInCents int64,
	futuresType constants.FuturesType) map[string]interface{} {

	side := "Buy"
	positionIdx := 1
	if futuresType == constants.SHORT {
		side = "Sell"
		positionIdx = 2
	}

	return api.buildFuturesParams(coin, amount, side, positionIdx)
}

func (api *BybitApi) buildCloseFuturesParams(coin *domains.Coin, openedTransaction *domains.Transaction, priceInCents int64) map[string]interface{} {
	side := "Sell"
	positionIdx := 1
	if openedTransaction.FuturesType == constants.SHORT {
		side = "Buy"
		positionIdx = 2
	}

	return api.buildFuturesParams(coin, openedTransaction.Amount, side, positionIdx)
}

func (api *BybitApi) buildFuturesParams(coin *domains.Coin, amount float64, side string, positionIdx int) map[string]interface{} {
	return map[string]interface{}{
		"api_key":          os.Getenv("BYBIT_CryptoBotFutures_API_KEY"),
		"qty":              amount,
		"side":             side,
		"symbol":           coin.Symbol,
		"timestamp":        util.MakeTimestamp(),
		"order_type":       "Market",
		"time_in_force":    "GoodTillCancel",
		"reduce_only":      false,
		"close_on_trigger": false,
		"position_idx":     positionIdx,
	}
}

func (api *BybitApi) futuresOrderByMarket(queryParams map[string]interface{}) (api.OrderResponseDto, error) {
	body, err := api.postSignedApiRequest("/private/linear/order/create", queryParams)
	if err != nil {
		return nil, err
	}

	dto := bybit.FuturesOrderResponseDto{}
	errUnmarshal := json.Unmarshal(body, &dto)
	if errUnmarshal != nil {
		zap.S().Error("Unmarshal error: ", errUnmarshal.Error())
		return nil, errUnmarshal
	}

	return &dto, nil
}
