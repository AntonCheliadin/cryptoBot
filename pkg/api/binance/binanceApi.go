package binance

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"go.uber.org/zap"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"tradingBot/pkg/api"
	"tradingBot/pkg/data/domains"
	"tradingBot/pkg/data/dto/binance"
	"tradingBot/pkg/util"
)

func NewBinanceApi() api.ExchangeApi {
	return &BinanceApi{}
}

type BinanceApi struct {
}

func (api *BinanceApi) GetCurrentCoinPrice(coin *domains.Coin) (int64, error) {
	resp, err := http.Get("https://api.binance.com/api/v3/ticker/price?symbol=" + coin.Symbol)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	var priceDto binance.PriceDto
	if err := json.NewDecoder(resp.Body).Decode(&priceDto); err != nil {
		return 0, err
	}

	return priceDto.PriceInCents()
}

func (api *BinanceApi) BuyCoinByMarket(coin *domains.Coin, amount float64) (api.OrderDto, error) {
	queryParams := api.buildParams(coin, amount, "BUY")
	return api.orderCoinByMarket(queryParams)
}

func (api *BinanceApi) SellCoinByMarket(coin *domains.Coin, amount float64) (api.OrderDto, error) {
	queryParams := api.buildParams(coin, amount, "SELL")
	return api.orderCoinByMarket(queryParams)
}

func (api *BinanceApi) orderCoinByMarket(queryParams string) (api.OrderDto, error) {
	zap.S().Infof("OrderCoinByMarket = %s", queryParams)

	uri := "https://api.binance.com/api/v3/order?" // /test
	signatureParameter := "&signature=" + api.sign(queryParams)

	url := uri + queryParams + signatureParameter

	method := "POST"

	client := &http.Client{}
	req, err := http.NewRequest(method, url, nil)

	if err != nil {
		zap.S().Errorf("API error: %s", err)
		return nil, err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("X-MBX-APIKEY", os.Getenv("BINANCE_API_KEY"))

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

	dto := binance.OrderResponseDto{}
	errUnmarshal := json.Unmarshal(body, &dto)
	if errUnmarshal != nil {
		zap.S().Error("Unmarshal error", errUnmarshal.Error())
		return nil, errUnmarshal
	}

	return &dto, nil
}

func (api *BinanceApi) buildParams(coin *domains.Coin, amount float64, side string) string {
	return "symbol=" + coin.Symbol +
		"&side=" + side +
		"&type=MARKET" +
		"&quantity=" + strings.TrimRight(fmt.Sprintf("%f", amount), "0") +
		"&timestamp=" + util.MakeTimestamp()
}

func (api *BinanceApi) sign(data string) string {
	secret := os.Getenv("BINANCE_SECRET_KEY")

	// Create a new HMAC by defining the hash type and the key (as byte array)
	h := hmac.New(sha256.New, []byte(secret))

	// Write Data to it
	h.Write([]byte(data))

	// Get result and encode as hexadecimal string
	sha := hex.EncodeToString(h.Sum(nil))

	return sha
}
