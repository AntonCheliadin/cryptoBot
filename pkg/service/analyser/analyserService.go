package analyser

import (
	"cryptoBot/pkg/api"
	"cryptoBot/pkg/data/domains"
	"cryptoBot/pkg/repository"
	"cryptoBot/pkg/service/trading"
	"cryptoBot/pkg/util"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
)

var analyserServiceImpl *AnalyserService

func NewAnalyserService(transactionRepo repository.Transaction, priceChangeRepo repository.PriceChange,
	exchangeApi api.ExchangeApi, tradingService *trading.HolderStrategyTradingService) *AnalyserService {
	if analyserServiceImpl != nil {
		panic("Unexpected try to create second service instance")
	}
	analyserServiceImpl = &AnalyserService{
		transactionRepo: transactionRepo,
		priceChangeRepo: priceChangeRepo,
		exchangeApi:     exchangeApi,
		tradingService:  tradingService,
	}
	return analyserServiceImpl
}

type AnalyserService struct {
	transactionRepo repository.Transaction
	priceChangeRepo repository.PriceChange
	exchangeApi     api.ExchangeApi
	tradingService  *trading.HolderStrategyTradingService
}

func (s *AnalyserService) AnalyseCoin(coin *domains.Coin, from string, to string, interval string) {
	prices := s.fetchAllBars(coin, from, to, interval)

	for _, price := range prices {
		s.tradingService.BotActionForPrice(coin, price)
	}
}

func (s *AnalyserService) fetchAllBars(coin *domains.Coin, from string, to string, interval string) []int64 {
	startMillis := fmt.Sprintf("%v", (util.GetMillisByDate(from)))
	endMillis := util.GetMillisByDate(to)

	prices := []int64{}

	for true {
		fmt.Printf("doing binance call startMillis %s \n", startMillis)
		url := "https://api.binance.com/api/v3/klines?symbol=" + coin.Symbol + "&interval=" + interval + "&limit=1000&startTime=" + startMillis
		method := "GET"

		client := &http.Client{}
		req, err := http.NewRequest(method, url, nil)

		if err != nil {
			fmt.Println(err)
			panic(err)
		}
		req.Header.Add("Content-Type", "application/json")

		res, err := client.Do(req)
		if err != nil {
			fmt.Println(err)
			panic(err)
		}
		defer res.Body.Close()

		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			fmt.Println(err)
			panic(err)
		}

		bars := string(body)

		split := strings.Split(bars, ",")

		for i := 1; i < len(split); i += 12 {
			priceInQuotes := split[i]
			startMillis = split[i+5]
			var price = strings.ReplaceAll(priceInQuotes, "\"", "")
			float, _ := strconv.ParseFloat(price, 64)
			prices = append(prices, int64(float*100))

			parseInt, _ := strconv.ParseInt(split[i+5], 10, 64)
			if parseInt > endMillis {
				return prices
			}
		}

	}

	return prices
}
