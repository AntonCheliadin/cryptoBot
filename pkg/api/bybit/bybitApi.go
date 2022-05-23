package bybit

import (
	"crypto/hmac"
	"crypto/sha256"
	"cryptoBot/pkg/api"
	"cryptoBot/pkg/constants"
	"cryptoBot/pkg/data/domains"
	"cryptoBot/pkg/data/dto/bybit"
	"cryptoBot/pkg/util"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"io/ioutil"
	"net/http"
	"os"
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
	body, err := api.postSignedApiRequest("/spot/v1/order?", queryParams)
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

func (api *BybitApi) postSignedApiRequest(uri string, queryParams string) ([]byte, error) {
	signatureParameter := "&sign=" + api.sign(queryParams)

	url := "https://api.bytick.com" + uri + queryParams + signatureParameter

	zap.S().Infof("OrderCoinByMarket = %s", url)

	method := "POST"

	client := &http.Client{}
	req, err := http.NewRequest(method, url, nil)

	if err != nil {
		zap.S().Errorf("API error: %s", err)
		return nil, err
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

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
	queryParams := "api_key=" + os.Getenv("BYBIT_CryptoBotSubAcc_API_KEY") +
		"&orderId=" + orderResponseDto.Result.OrderId +
		"&timestamp=" + util.MakeTimestamp()

	body, err := api.postSignedApiRequest("/spot/v1/history-orders?", queryParams)
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
	secret := os.Getenv("BYBIT_CryptoBotSubAcc_API_SECRET")

	// Create a new HMAC by defining the hash type and the key (as byte array)
	h := hmac.New(sha256.New, []byte(secret))

	// Write Data to it
	h.Write([]byte(data))

	// Get result and encode as hexadecimal string
	sha := hex.EncodeToString(h.Sum(nil))

	return sha
}

func (api *BybitApi) OpenFuturesOrder(coin *domains.Coin, amount float64, price int64, futuresType constants.FuturesType, leverage int) (api.OrderResponseDto, error) {
	return nil, errors.New("Futures api is not implemented")
}
func (api *BybitApi) CloseFuturesOrder(openedTransaction *domains.Transaction, price int64) (api.OrderResponseDto, error) {
	return nil, errors.New("Futures api is not implemented")
}
