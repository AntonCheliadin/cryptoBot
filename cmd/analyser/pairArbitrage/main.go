package main

import (
	"cryptoBot/cmd/analyser"
	"cryptoBot/cmd/bootstrap"
	"cryptoBot/pkg/api/bybit/mock"
	"cryptoBot/pkg/constants"
	"cryptoBot/pkg/log"
	"cryptoBot/pkg/repository"
	"cryptoBot/pkg/service/date"
	"cryptoBot/pkg/service/exchange"
	"cryptoBot/pkg/service/indicator/techanLib"
	"cryptoBot/pkg/service/orders"
	"cryptoBot/pkg/service/trading"
	"go.uber.org/zap"
	"os"
	"time"
)

func main() {
	bootstrap.Run()
	log.InitLoggerAnalyser()

	var closableClosure []func()

	defer func() {
		for i := range closableClosure {
			closableClosure[i]()
		}
	}()
	postgresDb := bootstrap.Database(closableClosure)
	repos := repository.NewRepositories(postgresDb)
	mockExchangeApi := mock.NewBybitApiMock()
	clockMock := date.GetClockMock()

	seriesConvertorService := techanLib.NewTechanConvertorService(clockMock, repos.Kline)
	exchangeDataService := exchange.NewExchangeDataService(repos.Transaction, repos.Coin, mockExchangeApi, clockMock, repos.Kline)
	priceChangeTrackingService := orders.NewPriceChangeTrackingService(repos.PriceChange)

	orderManagerService := orders.NewOrderManagerService(repos.Transaction, mockExchangeApi, clockMock, exchangeDataService, repos.Kline, constants.PAIR_ARBITRAGE, priceChangeTrackingService,
		orders.NewProfitLossFinderService(clockMock, repos.Kline),
		0,
		0, 0, 0, 0)

	klineInterval := 60

	var arguments = make([][]string, 0, 40)

	arguments = append(arguments, []string{"XRPUSDT", "LTCUSDT", "2023-06-09", "2023-07-14"})
	//arguments = append(arguments, []string{"XMRUSDT", "LTCUSDT", "2022-02-10", "2023-06-08"})
	//arguments = append(arguments, []string{"BNBUSDT", "ADAUSDT", "2021-07-12", "2023-05-30"})
	//arguments = append(arguments, []string{"ATOMUSDT", "BNBUSDT", "2021-11-01", "2023-06-09"})
	//arguments = append(arguments, []string{"AVAXUSDT", "DOTUSDT", "2021-09-25", "2023-06-13"})
	//
	//arguments = append(arguments, []string{"SOLUSDT", "UNIUSDT", "2021-07-10", "2023-06-14"})
	//arguments = append(arguments, []string{"SOLUSDT", "NEARUSDT", "2021-10-25", "2023-06-14"})
	//arguments = append(arguments, []string{"DASHUSDT", "IMXUSDT", "2021-12-10", "2023-06-14"})
	//arguments = append(arguments, []string{"ZECUSDT", "NEARUSDT", "2021-12-10", "2023-06-14"})
	//arguments = append(arguments, []string{"ALGOUSDT", "NEARUSDT", "2021-10-25", "2023-06-14"})
	//arguments = append(arguments, []string{"DASHUSDT", "ALGOUSDT", "2021-10-25", "2023-06-14"})
	//arguments = append(arguments, []string{"ZECUSDT", "FILUSDT", "2021-12-10", "2023-06-14"})
	//arguments = append(arguments, []string{"IMXUSDT", "DYDXUSDT", "2021-12-10", "2023-06-14"})
	//arguments = append(arguments, []string{"UNIUSDT", "MATICUSDT", "2021-07-10", "2023-06-14"})
	//arguments = append(arguments, []string{"FLOWUSDT", "FILUSDT", "2021-12-10", "2023-06-14"})
	//
	//arguments = append(arguments, []string{"BTCUSDT", "ADAUSDT", "2022-02-10", "2023-01-06"})
	//arguments = append(arguments, []string{"BTCUSDT", "ETHUSDT", "2022-02-10", "2023-05-31"})
	//
	//arguments = append(arguments, []string{"ZECUSDT", "XMRUSDT", "2022-02-10", "2023-05-31"})
	//arguments = append(arguments, []string{"AVAXUSDT", "SOLUSDT", "2021-10-10", "2023-05-31"})
	//arguments = append(arguments, []string{"NEARUSDT", "DOTUSDT", "2021-10-25", "2023-05-31"})
	//arguments = append(arguments, []string{"NEARUSDT", "ATOMUSDT", "2021-10-25", "2023-05-31"}) //800
	//arguments = append(arguments, []string{"AVAXUSDT", "ATOMUSDT", "2021-10-25", "2023-05-31"})
	//arguments = append(arguments, []string{"BNBUSDT", "SOLUSDT", "2021-07-10", "2023-05-31"})
	//arguments = append(arguments, []string{"BTCUSDT", "XMRUSDT", "2022-02-10", "2023-05-31"})
	//
	//arguments = append(arguments, []string{"BNBUSDT", "ATOMUSDT", "2022-01-10", "2023-05-31"})
	//arguments = append(arguments, []string{"SOLUSDT", "DOTUSDT", "2022-01-10", "2023-05-31"})
	//arguments = append(arguments, []string{"AVAXUSDT", "NEARUSDT", "2022-01-10", "2023-05-31"}) //806
	//arguments = append(arguments, []string{"XMRUSDT", "SOLUSDT", "2022-02-10", "2023-05-31"})
	//arguments = append(arguments, []string{"XMRUSDT", "ETHUSDT", "2022-02-10", "2023-05-31"})

	for _, argument := range arguments {
		symbol1 := argument[0]
		symbol2 := argument[1]
		from := argument[2]
		to := argument[3]

		println(symbol1, symbol2, from, to)

		coin1, _ := repos.Coin.FindBySymbol(symbol1)
		coin2, _ := repos.Coin.FindBySymbol(symbol2)

		tradingService := trading.NewPairArbitrageStrategyTradingService(
			repos.Coin,
			repos.Transaction,
			clockMock,
			exchangeDataService,
			repos.SyntheticKline,
			exchange.NewKlinesFetcherService(mockExchangeApi, repos.Kline, clockMock),
			orderManagerService,
			seriesConvertorService,
			coin1,
			coin2,
		)
		analyserService := analyser.NewAnalyserRunner(tradingService)

		start := time.Now()
		analyserService.AnalyseCoin(from, to, klineInterval)

		end := time.Now()
		zap.S().Infof("EXECUTED %s-%s  in %s", symbol1, symbol2, end.Sub(start).Milliseconds())

		//tradingStrategy := 780 + i //770-780 fail
		//postgresDb.Exec("update transaction_table set trading_strategy = $1 where trading_strategy = 6;", tradingStrategy)
	}

	if err := postgresDb.Close(); err != nil {
		zap.S().Errorf("error occured on db connection close: %s", err.Error())
	}

	os.Exit(0)
}
