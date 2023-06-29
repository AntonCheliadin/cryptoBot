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

	var arguments = make([][]string, 0, 10)

	arguments = append(arguments, []string{"XRPUSDT", "LTCUSDT", "2021-06-02", "2023-06-09"})
	//arguments = append(arguments, []string{"XMRUSDT", "LTCUSDT", "2022-02-02", "2023-06-08"})
	arguments = append(arguments, []string{"BNBUSDT", "ADAUSDT", "2021-07-02", "2023-05-30"})
	//arguments = append(arguments, []string{"ATOMUSDT", "BNBUSDT", "2021-10-23", "2023-06-09"})
	//arguments = append(arguments, []string{"AVAXUSDT", "DOTUSDT", "2021-09-18", "2023-06-13"})

	//arguments = append(arguments, []string{"SOLUSDT", "UNIUSDT", "2021-07-01", "2023-06-14"})   //650 -201%
	//arguments = append(arguments, []string{"SOLUSDT", "NEARUSDT", "2021-10-15", "2023-06-14"})  //651 -81%
	//arguments = append(arguments, []string{"DASHUSDT", "IMXUSDT", "2021-12-01", "2023-06-14"})  //660 total profit 107%
	//arguments = append(arguments, []string{"ZECUSDT", "NEARUSDT", "2021-12-01", "2023-06-14"})  //661 -39%
	arguments = append(arguments, []string{"ALGOUSDT", "NEARUSDT", "2021-10-15", "2023-06-14"}) //662 total profit 282%
	arguments = append(arguments, []string{"DASHUSDT", "ALGOUSDT", "2021-10-13", "2023-06-14"}) //680 225%
	//arguments = append(arguments, []string{"ZECUSDT", "FILUSDT", "2021-12-01", "2023-06-14"})   //682 5%
	arguments = append(arguments, []string{"IMXUSDT", "DYDXUSDT", "2021-12-01", "2023-06-14"})  //673 185%
	arguments = append(arguments, []string{"UNIUSDT", "MATICUSDT", "2021-07-01", "2023-06-14"}) //671 119%
	arguments = append(arguments, []string{"FLOWUSDT", "FILUSDT", "2021-12-01", "2023-06-14"})  //672  64%

	arguments = append(arguments, []string{"BTCUSDT", "ADAUSDT", "2022-02-01", "2023-01-06"})
	arguments = append(arguments, []string{"BTCUSDT", "ETHUSDT", "2022-02-03", "2023-05-31"})

	arguments = append(arguments, []string{"ZECUSDT", "XMRUSDT", "2022-02-01", "2023-05-31"})
	arguments = append(arguments, []string{"AVAXUSDT", "SOLUSDT", "2021-10-01", "2023-05-31"})
	arguments = append(arguments, []string{"NEARUSDT", "DOTUSDT", "2021-10-15", "2023-05-31"})
	//arguments = append(arguments, []string{"NEARUSDT", "ATOMUSDT", "2021-10-15", "2023-05-31"})
	arguments = append(arguments, []string{"AVAXUSDT", "ATOMUSDT", "2021-10-15", "2023-05-31"})
	//arguments = append(arguments, []string{"BNBUSDT", "SOLUSDT", "2021-07-01", "2023-05-31"})
	arguments = append(arguments, []string{"BTCUSDT", "XMRUSDT", "2022-02-01", "2023-05-31"})

	arguments = append(arguments, []string{"BNBUSDT", "ATOMUSDT", "2022-01-01", "2023-05-31"})
	arguments = append(arguments, []string{"SOLUSDT", "DOTUSDT", "2022-01-01", "2023-05-31"})
	arguments = append(arguments, []string{"AVAXUSDT", "NEARUSDT", "2022-01-01", "2023-05-31"})
	arguments = append(arguments, []string{"XMRUSDT", "SOLUSDT", "2022-02-01", "2023-05-31"})
	arguments = append(arguments, []string{"XMRUSDT", "ETHUSDT", "2022-02-01", "2023-05-31"})

	for i, argument := range arguments {
		symbol1 := argument[0]
		symbol2 := argument[1]
		from := argument[2]
		to := argument[3]

		println(symbol1, symbol2, from, to)

		coin1, _ := repos.Coin.FindBySymbol(symbol1)
		coin2, _ := repos.Coin.FindBySymbol(symbol2)

		tradingService := trading.NewPairArbitrageStrategyTradingService(
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

		tradingStrategy := 750 + i
		postgresDb.Exec("update transaction_table set trading_strategy = $1 where trading_strategy = 6;", tradingStrategy)
	}

	if err := postgresDb.Close(); err != nil {
		zap.S().Errorf("error occured on db connection close: %s", err.Error())
	}

	os.Exit(0)
}
