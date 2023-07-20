package bybit

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"cryptoBot/pkg/api"
	"cryptoBot/pkg/constants"
	"cryptoBot/pkg/constants/futureType"
	"cryptoBot/pkg/data/domains"
	"cryptoBot/pkg/data/dto/bybit"
	"cryptoBot/pkg/data/dto/bybit/order"
	"cryptoBot/pkg/data/dto/bybit/position"
	"cryptoBot/pkg/data/dto/bybit/wallet"
	"cryptoBot/pkg/util"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"go.uber.org/zap"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func NewBybitApi(apiKey string, secretKey string) api.ExchangeApi {
	return &BybitApi{
		apiKey:    apiKey,
		secretKey: secretKey,
	}
}

type BybitApi struct {
	apiKey    string
	secretKey string
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
func (bybitApi *BybitApi) GetKlinesFutures(coin *domains.Coin, interval string, limit int, fromTime time.Time) (api.KlinesDto, error) {
	intervalInt, _ := strconv.Atoi(interval)
	end := fromTime.Add(time.Minute * time.Duration(intervalInt*limit))

	resp, err := http.Get("https://api.bytick.com/derivatives/v3/public/kline?" +
		"category=linear" +
		"&symbol=" + coin.Symbol +
		"&interval=" + interval +
		"&start=" + strconv.FormatInt(util.GetMillisByTime(fromTime), 10) +
		"&end=" + strconv.FormatInt(util.GetMillisByTime(end), 10) +
		"&limit=" + strconv.Itoa(limit))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	dto := &bybit.KlinesFuturesDto{Interval: intervalInt}
	if err := json.NewDecoder(resp.Body).Decode(&dto); err != nil {
		return nil, err
	}

	return dto, nil
}

func (api *BybitApi) GetCurrentCoinPriceForFutures(coin *domains.Coin) (float64, error) {
	resp, err := http.Get("https://api.bytick.com/derivatives/v3/public/tickers?symbol=" + coin.Symbol)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	var priceDto bybit.TickerInfoDto
	if err := json.NewDecoder(resp.Body).Decode(&priceDto); err != nil {
		return 0, err
	}

	return priceDto.Price()
}

func (api *BybitApi) GetCurrentCoinPrice(coin *domains.Coin) (float64, error) {
	resp, err := http.Get("https://api.bytick.com/spot/quote/v1/ticker/price?symbol=" + coin.Symbol)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	var priceDto bybit.PriceDto
	if err := json.NewDecoder(resp.Body).Decode(&priceDto); err != nil {
		return 0, err
	}

	return priceDto.Price()
}

func (api *BybitApi) BuyCoinByMarket(coin *domains.Coin, amount float64, price float64) (api.OrderResponseDto, error) {
	queryParams := api.buildParams(coin, amount, price, "Buy")
	return api.orderCoinByMarket(queryParams)
}

func (api *BybitApi) SellCoinByMarket(coin *domains.Coin, amount float64, price float64) (api.OrderResponseDto, error) {
	queryParams := api.buildParams(coin, amount, price, "Sell")
	return api.orderCoinByMarket(queryParams)
}

func (api *BybitApi) orderCoinByMarket(queryParams map[string]interface{}) (api.OrderResponseDto, error) {
	body, err := api.postSignedApiRequest("/spot/v1/order", queryParams)
	if err != nil {
		return nil, err
	}

	dto := order.OrderResponseDto{}
	errUnmarshal := json.Unmarshal(body, &dto)
	if errUnmarshal != nil {
		zap.S().Error("Unmarshal error", errUnmarshal.Error())
		return nil, errUnmarshal
	}

	if dto.RetCode != 0 {
		return nil, errors.New(dto.RetMsg)
	}

	time.Sleep(30 * time.Second)

	return api.getSpotTradeHistory(dto)
}

func (api *BybitApi) getSignedApiRequest(uri string, queryParams map[string]interface{}) ([]byte, error) {
	sign := api.getSignature(queryParams)
	url := uri + "?" + util.ConvertMapParamsToString(queryParams) + "&sign=" + sign

	return api.signedApiRequest(http.MethodGet, url, nil)
}

func (api *BybitApi) postSignedApiRequest(uri string, queryParams map[string]interface{}) ([]byte, error) {
	queryParams["sign"] = api.getSignature(queryParams)
	jsonString, _ := json.Marshal(queryParams)

	return api.signedApiRequest(http.MethodPost, uri, bytes.NewBuffer(jsonString))
}

func (api *BybitApi) signedApiRequest(method, uri string, requestBody io.Reader) ([]byte, error) {
	urlRequest := "https://api.bytick.com" + uri
	client := &http.Client{}
	req, err := http.NewRequest(method, urlRequest, requestBody)

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
	return body, nil
}

func (api *BybitApi) getSpotOrderDetails(orderResponseDto order.OrderResponseDto) (api.OrderResponseDto, error) {
	requestParams := map[string]interface{}{
		"api_key":   api.apiKey,
		"order_id":  orderResponseDto.Result.OrderId,
		"timestamp": util.MakeTimestamp(),
		"symbol":    orderResponseDto.Result.Symbol,
	}

	body, err := api.getSignedApiRequest("/spot/v1/history-orders", requestParams)
	if err != nil {
		return nil, err
	}

	dto := order.OrderHistoryDto{}
	errUnmarshal := json.Unmarshal(body, &dto)
	if errUnmarshal != nil {
		zap.S().Error("Unmarshal error", errUnmarshal.Error())
		return nil, errUnmarshal
	}

	return &dto, nil
}

func (api *BybitApi) getSpotTradeHistory(orderResponseDto order.OrderResponseDto) (api.OrderResponseDto, error) {
	requestParams := map[string]interface{}{
		"api_key":   api.apiKey,
		"orderId":   orderResponseDto.Result.OrderId,
		"timestamp": util.MakeTimestamp(),
		"symbol":    orderResponseDto.Result.Symbol,
	}

	body, err := api.getSignedApiRequest("/spot/v1/myTrades", requestParams)
	if err != nil {
		return nil, err
	}

	dto := order.TradeHistoryDto{}
	errUnmarshal := json.Unmarshal(body, &dto)
	if errUnmarshal != nil {
		zap.S().Error("Unmarshal error", errUnmarshal.Error())
		return nil, errUnmarshal
	}

	return &dto, nil
}

func (api *BybitApi) buildParams(coin *domains.Coin, amount float64, price float64, side string) map[string]interface{} {
	return map[string]interface{}{
		"api_key":   api.apiKey,
		"qty":       api.buildQty(amount, price, side),
		"side":      side,
		"symbol":    coin.Symbol,
		"timestamp": util.MakeTimestamp(),
		"type":      "MARKET",
	}
}

/**
https://bybit-exchange.github.io/docs/spot/v1/#t-placeactive
Order quantity
for market orders: when side is Buy, this is in the quote currency.
Otherwise, qty is in the base currency.
For example, on BTCUSDT a Buy order is in USDT, otherwise it's in BTC. For limit orders, the qty is always in the base currency.
*/
func (api *BybitApi) buildQty(amount float64, price float64, side string) string {
	if side == "Buy" {
		costInUsd := price
		return strings.TrimRight(fmt.Sprintf("%f", amount*costInUsd), "0")
	} else {
		return strings.TrimRight(fmt.Sprintf("%f", amount), "0")
	}
}

func (api *BybitApi) sign(data string) string {
	// Create a new HMAC by defining the hash type and the key (as byte array)
	h := hmac.New(sha256.New, []byte(api.secretKey))

	// Write Data to it
	h.Write([]byte(data))

	// Get result and encode as hexadecimal string
	sha := hex.EncodeToString(h.Sum(nil))

	return sha
}

func (api *BybitApi) getSignature(params map[string]interface{}) string {
	h := hmac.New(sha256.New, []byte(api.secretKey))
	io.WriteString(h, util.ConvertMapParamsToString(params))
	return fmt.Sprintf("%x", h.Sum(nil))
}

func (api *BybitApi) SetFuturesLeverage(coin *domains.Coin, leverage int) error {
	_, err := api.postSignedApiRequest("/private/linear/position/set-leverage",
		map[string]interface{}{
			"api_key":       api.apiKey,
			"buy_leverage":  strconv.Itoa(leverage),
			"sell_leverage": strconv.Itoa(leverage),
			"symbol":        coin.Symbol,
			"timestamp":     util.MakeTimestamp(),
		},
	)

	return err
}

func (api *BybitApi) SetIsolatedMargin(coin *domains.Coin, leverage int) error {
	_, err := api.postSignedApiRequest("/contract/v3/private/position/switch-isolated",
		map[string]interface{}{
			"api_key":       api.apiKey,
			"tradeMode":     1, //0: cross margin. 1: isolated margin
			"buy_leverage":  strconv.Itoa(leverage),
			"sell_leverage": strconv.Itoa(leverage),
			"symbol":        coin.Symbol,
			"timestamp":     util.MakeTimestamp(),
		},
	)

	return err
}

func (api *BybitApi) OpenFuturesOrder(coin *domains.Coin, amount float64, price float64, futuresType futureType.FuturesType, stopLossPriceInCents float64) (api.OrderResponseDto, error) {
	queryParams := api.buildOpenFuturesParams(coin, amount, price, futuresType, stopLossPriceInCents)
	return api.futuresOrderByMarketWithResponseDetails(queryParams)
}

func (api *BybitApi) CloseFuturesOrder(coin *domains.Coin, openedTransaction *domains.Transaction, price float64) (api.OrderResponseDto, error) {
	queryParams := api.buildCloseFuturesParams(coin, openedTransaction, price)
	return api.futuresOrderByMarketWithResponseDetails(queryParams)
}

func (api *BybitApi) buildOpenFuturesParams(coin *domains.Coin, amount float64, price float64,
	futuresType futureType.FuturesType, stopLossPriceInCents float64) map[string]interface{} {

	side := "Buy"
	positionIdx := 1
	if futuresType == futureType.SHORT {
		side = "Sell"
		positionIdx = 2
	}

	requestParams := api.buildFuturesParams(coin, amount, side, positionIdx)

	if stopLossPriceInCents > 0 {
		requestParams["stop_loss"] = stopLossPriceInCents
	}

	return requestParams
}

func (api *BybitApi) buildCloseFuturesParams(coin *domains.Coin, openedTransaction *domains.Transaction, price float64) map[string]interface{} {
	side := "Sell"
	positionIdx := 1
	if openedTransaction.FuturesType == futureType.SHORT {
		side = "Buy"
		positionIdx = 2
	}

	return api.buildFuturesParams(coin, openedTransaction.Amount, side, positionIdx)
}

func (api *BybitApi) buildFuturesParams(coin *domains.Coin, amount float64, side string, positionIdx int) map[string]interface{} {
	return map[string]interface{}{
		"api_key":          api.apiKey,
		"recv_window":      60000,
		"qty":              amount,
		"side":             side,
		"symbol":           coin.Symbol,
		"timestamp":        util.MakeTimestamp(),
		"order_link_id":    coin.Symbol + "-" + time.Now().Format(constants.DATE_TIME_FORMAT),
		"order_type":       "Market",
		"time_in_force":    "GoodTillCancel",
		"reduce_only":      false,
		"close_on_trigger": false,
		"position_idx":     positionIdx,
	}
}

func (api *BybitApi) futuresOrderByMarket(queryParams map[string]interface{}) (*order.FuturesOrderResponseDto, error) {
	body, err := api.postSignedApiRequest("/private/linear/order/create", queryParams)
	if err != nil {
		return nil, err
	}

	zap.S().Debugf("API response: %s", string(body))

	dto := order.FuturesOrderResponseDto{}
	errUnmarshal := json.Unmarshal(body, &dto)
	if errUnmarshal != nil {
		zap.S().Error("Unmarshal error: ", errUnmarshal.Error())
		return nil, errUnmarshal
	}

	if dto.RetCode != 0 {
		return nil, errors.New("Create order failed!")
	}

	return &dto, nil
}

func (api *BybitApi) futuresOrderByMarketWithResponseDetails(queryParams map[string]interface{}) (api.OrderResponseDto, error) {
	dto, err := api.futuresOrderByMarket(queryParams)
	if err != nil {
		return nil, err
	}

	for i := 0; i < 10; i++ {
		time.Sleep(10 * time.Second)

		responseDto, err := api.GetActiveOrder(dto)
		if err == nil {
			return responseDto, nil
		}
	}
	return api.futuresOrderByMarketWithResponseDetails(queryParams)
}

func (api *BybitApi) IsFuturesPositionOpened(coin *domains.Coin, openedOrder *domains.Transaction) bool {
	positionDto, err := api.GetPosition(coin)
	if err != nil {
		zap.S().Error("Error on getting position", err.Error())
		return true
	}

	for _, positionDto := range positionDto.Result {
		if "Buy" == positionDto.Side && openedOrder.FuturesType == futureType.LONG ||
			"Sell" == positionDto.Side && openedOrder.FuturesType == futureType.SHORT {
			if positionDto.Size > 0 {
				zap.S().Infof("Position side=%s unrealizedPNL=%vUSDT", positionDto.Side, positionDto.UnrealisedPnl)
			}
			return positionDto.Size > 0
		}
	}
	zap.S().Error("Error on searching position")
	return true
}

func (api *BybitApi) GetLastFuturesOrder(coin *domains.Coin, clientOrderId string) (api.OrderResponseDto, error) {
	requestParams := map[string]interface{}{
		"api_key":   api.apiKey,
		"order_id":  clientOrderId,
		"timestamp": util.MakeTimestamp(),
		"symbol":    coin.Symbol,
	}

	body, err := api.getSignedApiRequest("/private/linear/order/list", requestParams)
	if err != nil {
		return nil, err
	}

	dto := order.ActiveOrdersResponseDto{}
	errUnmarshal := json.Unmarshal(body, &dto)
	if errUnmarshal != nil {
		zap.S().Error("Unmarshal error", errUnmarshal.Error())
		return nil, errUnmarshal
	}

	if len(dto.Result.Data) > 0 {
		return &dto.Result.Data[0], nil
	}

	return nil, nil
}

func (api *BybitApi) GetActiveFuturesConditionalOrder(coin *domains.Coin, conditionalOrder *domains.ConditionalOrder) (api.OrderResponseDto, error) {
	requestParams := map[string]interface{}{
		"api_key":   api.apiKey,
		"order_id":  conditionalOrder.ClientOrderId.String,
		"timestamp": util.MakeTimestamp(),
		"symbol":    coin.Symbol,
	}

	body, err := api.getSignedApiRequest("/private/linear/stop-order/list", requestParams)
	if err != nil {
		return nil, err
	}

	dto := order.ActiveOrdersResponseDto{}
	errUnmarshal := json.Unmarshal(body, &dto)
	if errUnmarshal != nil {
		zap.S().Error("Unmarshal error", errUnmarshal.Error())
		return nil, errUnmarshal
	}

	if len(dto.Result.Data) > 0 {
		return &dto.Result.Data[0], nil
	}

	return nil, nil
}

func (api *BybitApi) GetFuturesActiveOrdersByCoin(coin *domains.Coin) (*order.ActiveOrdersResponseDto, error) {
	requestParams := map[string]interface{}{
		"api_key":   api.apiKey,
		"timestamp": util.MakeTimestamp(),
		"symbol":    coin.Symbol,
	}

	body, err := api.getSignedApiRequest("/private/linear/order/list", requestParams)
	if err != nil {
		return nil, err
	}

	dto := order.ActiveOrdersResponseDto{}
	errUnmarshal := json.Unmarshal(body, &dto)
	if errUnmarshal != nil {
		zap.S().Error("Unmarshal error", errUnmarshal.Error())
		return nil, errUnmarshal
	}

	return &dto, nil
}

func (api *BybitApi) GetActiveOrder(orderDto *order.FuturesOrderResponseDto) (api.OrderResponseDto, error) {
	requestParams := map[string]interface{}{
		"api_key":   api.apiKey,
		"order_id":  orderDto.Result.OrderId,
		"timestamp": util.MakeTimestamp(),
		"symbol":    orderDto.Result.Symbol,
	}

	body, err := api.getSignedApiRequest("/private/linear/order/list", requestParams)
	if err != nil {
		return nil, err
	}

	dto := order.ActiveOrdersResponseDto{}
	errUnmarshal := json.Unmarshal(body, &dto)
	if errUnmarshal != nil {
		zap.S().Error("Unmarshal error", errUnmarshal.Error())
		return nil, errUnmarshal
	}

	if len(dto.Result.Data) == 0 {
		return nil, errors.New("empty response")
	}

	return &dto.Result.Data[0], nil
}

func (api *BybitApi) GetWalletBalance() (api.WalletBalanceDto, error) {
	requestParams := map[string]interface{}{
		"api_key":   api.apiKey,
		"coin":      "USDT",
		"timestamp": util.MakeTimestamp(),
	}

	body, err := api.getSignedApiRequest("/v2/private/wallet/balance", requestParams)
	if err != nil {
		return nil, err
	}

	dto := wallet.GetWalletBalanceDto{}
	errUnmarshal := json.Unmarshal(body, &dto)
	if errUnmarshal != nil {
		zap.S().Error("Unmarshal error", errUnmarshal.Error())
		return nil, errUnmarshal
	}

	return &dto, nil
}

func (api *BybitApi) OpenFuturesConditionalOrder(coin *domains.Coin, amount float64, price int64, basePrice int64, stopPX int64, futuresType futureType.FuturesType) (api.OrderResponseDto, error) {
	side := "Buy"
	positionIdx := 1
	if futuresType == futureType.SHORT {
		side = "Sell"
		positionIdx = 2
	}

	queryParams := map[string]interface{}{
		"api_key":          api.apiKey,
		"qty":              amount,
		"side":             side,
		"symbol":           coin.Symbol,
		"timestamp":        util.MakeTimestamp(),
		"order_link_id":    coin.Symbol + "-" + time.Now().Format(constants.DATE_TIME_FORMAT),
		"order_type":       "Limit",
		"price":            util.GetDollarsByCents(price),
		"base_price":       util.GetDollarsByCents(basePrice), /*It will be used to compare with the value of stop_px, to decide whether your conditional order will be triggered by crossing trigger price from upper side or lower side. Mainly used to identify the expected direction of the current conditional order.*/
		"stop_px":          util.GetDollarsByCents(stopPX),    /*Trigger price. If you're expecting the price to rise to trigger your conditional order, make sure stop_px > max(market price, base_price) else, stop_px < min(market price, base_price)*/
		"time_in_force":    "GoodTillCancel",
		"trigger_by":       "LastPrice",
		"reduce_only":      false,
		"close_on_trigger": true,
		"position_idx":     positionIdx,
	}

	body, err := api.postSignedApiRequest("/private/linear/stop-order/create", queryParams)
	if err != nil {
		return nil, err
	}

	dto := order.FuturesOrderResponseDto{}
	errUnmarshal := json.Unmarshal(body, &dto)
	if errUnmarshal != nil {
		zap.S().Error("Unmarshal error: ", errUnmarshal.Error())
		return nil, errUnmarshal
	}

	if dto.RetCode != 0 {
		return nil, errors.New("Create order failed!")
	}

	return &dto, nil
}

func (api *BybitApi) GetConditionalOrder(coin *domains.Coin) (*order.GetConditionalOrderDto, error) {
	requestParams := map[string]interface{}{
		"api_key":   api.apiKey,
		"timestamp": util.MakeTimestamp(),
		"symbol":    coin.Symbol,
	}

	body, err := api.getSignedApiRequest("/private/linear/stop-order/list", requestParams)
	if err != nil {
		return nil, err
	}

	dto := order.GetConditionalOrderDto{}
	errUnmarshal := json.Unmarshal(body, &dto)
	if errUnmarshal != nil {
		zap.S().Error("Unmarshal error", errUnmarshal.Error())
		return nil, errUnmarshal
	}

	return &dto, nil
}

func (api *BybitApi) GetPosition(coin *domains.Coin) (*position.GetPositionDto, error) {
	requestParams := map[string]interface{}{
		"api_key":   api.apiKey,
		"symbol":    coin.Symbol,
		"timestamp": util.MakeTimestamp(),
	}

	body, err := api.getSignedApiRequest("/private/linear/position/list", requestParams)
	if err != nil {
		return nil, err
	}

	dto := position.GetPositionDto{}
	errUnmarshal := json.Unmarshal(body, &dto)
	if errUnmarshal != nil {
		zap.S().Error("Unmarshal error", errUnmarshal.Error())
		return nil, errUnmarshal
	}

	return &dto, nil
}

func (api *BybitApi) GetTradeRecords(coin *domains.Coin, openTransaction *domains.Transaction) (*position.GetTradeRecordsDto, error) {
	requestParams := map[string]interface{}{
		"api_key":    api.apiKey,
		"symbol":     coin.Symbol,
		"exec_type":  "Trade",
		"start_time": util.GetMillisByTime(openTransaction.CreatedAt),
		"timestamp":  util.MakeTimestamp(),
	}

	body, err := api.getSignedApiRequest("/private/linear/trade/execution/list", requestParams)
	if err != nil {
		return nil, err
	}

	dto := position.GetTradeRecordsDto{}
	errUnmarshal := json.Unmarshal(body, &dto)
	if errUnmarshal != nil {
		zap.S().Error("Unmarshal error", errUnmarshal.Error())
		return nil, errUnmarshal
	}

	return &dto, nil
}

func (api *BybitApi) GetCloseTradeRecord(coin *domains.Coin, openTransaction *domains.Transaction) (api.OrderResponseDto, error) {
	tradeRecordsDto, err := api.GetTradeRecords(coin, openTransaction)
	if err != nil {
		return nil, err
	}

	var trades []position.TradeRecordDto

	for _, tradeRecordDto := range tradeRecordsDto.Result.Data {
		if "Sell" == tradeRecordDto.Side && openTransaction.FuturesType == futureType.LONG ||
			"Buy" == tradeRecordDto.Side && openTransaction.FuturesType == futureType.SHORT {
			trades = append(trades, tradeRecordDto)
		}
	}

	tradesSummaryDto := position.TradesSummaryDto{Trades: trades}

	if tradesSummaryDto.GetAmount() != openTransaction.Amount {
		panic(fmt.Sprintf("Unexpected amount in trade records. Expected: %v; actual: %v", openTransaction.Amount, tradesSummaryDto.GetAmount()))
	}

	return &tradesSummaryDto, nil
}

func (api *BybitApi) ReplaceFuturesActiveOrder(coin *domains.Coin, transaction *domains.Transaction, stopLossPriceInCents int64) (*order.ReplaceFuturesActiveOrder, error) {
	queryParams := map[string]interface{}{
		"api_key":   api.apiKey,
		"order_id":  transaction.ClientOrderId.String,
		"symbol":    coin.Symbol,
		"stop_loss": util.GetDollarsByCents(stopLossPriceInCents),
		"timestamp": util.MakeTimestamp(),
	}

	body, err := api.postSignedApiRequest("/private/linear/order/replace", queryParams)
	if err != nil {
		return nil, err
	}

	dto := order.ReplaceFuturesActiveOrder{}
	errUnmarshal := json.Unmarshal(body, &dto)
	if errUnmarshal != nil {
		zap.S().Error("Unmarshal error: ", errUnmarshal.Error())
		return nil, errUnmarshal
	}

	if dto.RetCode != 0 {
		return nil, errors.New("Failed!")
	}

	return &dto, nil
}
