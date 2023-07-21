package chart

import (
	"bytes"
	"cryptoBot/pkg/data/dto/postgres/transaction"
	"cryptoBot/pkg/repository"
	moneyUtil "cryptoBot/pkg/util"
	"fmt"
	"github.com/wcharczuk/go-chart"
	"github.com/wcharczuk/go-chart/util"
	"go.uber.org/zap"
	"io/ioutil"
	"strconv"
	"time"
)

var chartTradingStrategyServiceImpl *ChartTradingStrategyService

func NewChartTradingStrategyService(transactionRepo repository.Transaction, coinRepo repository.Coin) *ChartTradingStrategyService {
	if chartTradingStrategyServiceImpl != nil {
		panic("Unexpected try to create second service instance")
	}
	chartTradingStrategyServiceImpl = &ChartTradingStrategyService{
		transactionRepo:   transactionRepo,
		coinRepo:          coinRepo,
		InitialWalletCost: 20000,
	}
	return chartTradingStrategyServiceImpl
}

type ChartTradingStrategyService struct {
	klineRepo         repository.Kline
	coinRepo          repository.Coin
	transactionRepo   repository.Transaction
	InitialWalletCost int64
}

func (s *ChartTradingStrategyService) ChartWalletTradingStrategy(tradingStrategy int) {
	profitPercents, err := s.transactionRepo.FindAllProfitPercents(tradingStrategy)
	if err != nil {
		zap.S().Errorf("Error during search profit %s", err.Error())
		return
	}
	if len(profitPercents) == 0 {
		return
	}

	chartName := s.buildChartName(tradingStrategy, " Wallet chart")

	xvalues, yvalues := s.collectWalletData(profitPercents)

	mainSeries := chart.TimeSeries{
		Name: chartName,
		Style: chart.Style{
			Show:        true,
			StrokeColor: chart.ColorBlue,
			FillColor:   chart.ColorBlue.WithAlpha(50),
		},
		XValues: xvalues,
		YValues: yvalues,
	}

	graph := chart.Chart{
		Width:  1500,
		Height: 500,
		Background: chart.Style{
			Padding: chart.Box{
				Top: 50,
			},
		},
		Series: []chart.Series{
			mainSeries,
		},
		XAxis: chart.XAxis{
			Style: chart.Style{
				Show: true,
			},
			TickPosition: chart.TickPositionBetweenTicks,
			ValueFormatter: func(v interface{}) string {
				typed := v.(float64)
				typedDate := util.Time.FromFloat64(typed)
				return fmt.Sprintf("%d.%d.%d", typedDate.Day(), typedDate.Month(), typedDate.Year())
			},
		},
		YAxis: chart.YAxis{
			Style: chart.Style{
				Show: true,
			},
		},
	}

	graph.Elements = []chart.Renderable{chart.LegendThin(&graph)}

	s.saveToFile(graph, chartName)
}

func (s *ChartTradingStrategyService) buildChartName(tradingStrategy int, chartType string) string {
	coinIds, _ := s.transactionRepo.FindAllCoinIds(tradingStrategy)
	coin1, _ := s.coinRepo.FindById(coinIds[0])
	coin2, _ := s.coinRepo.FindById(coinIds[1])

	chartName := coin1.Symbol + " - " + coin2.Symbol + " " + strconv.Itoa(tradingStrategy) + chartType
	return chartName
}

func (s *ChartTradingStrategyService) ChartTransactionsTradingStrategy(tradingStrategy int) {
	profitPercents, err := s.transactionRepo.FindAllProfitPercents(tradingStrategy)
	if err != nil {
		zap.S().Errorf("Error during search profit %s", err.Error())
		return
	}
	if len(profitPercents) == 0 {
		return
	}

	chartName := s.buildChartName(tradingStrategy, " Transactions chart")

	xvalues, yvalues := s.collectTransactionsData(profitPercents)

	mainSeries := chart.TimeSeries{
		Name: chartName,
		Style: chart.Style{
			Show:        true,
			StrokeColor: chart.ColorBlue,
			FillColor:   chart.ColorBlue.WithAlpha(50),
		},
		XValues: xvalues,
		YValues: yvalues,
	}

	graph := chart.Chart{
		Width:  1500,
		Height: 500,
		Background: chart.Style{
			Padding: chart.Box{
				Top: 50,
			},
		},
		Series: []chart.Series{
			mainSeries,
		},
		XAxis: chart.XAxis{
			Style: chart.Style{
				Show: true,
			},
			TickPosition: chart.TickPositionBetweenTicks,
			ValueFormatter: func(v interface{}) string {
				typed := v.(float64)
				typedDate := util.Time.FromFloat64(typed)
				return fmt.Sprintf("%d.%d.%d", typedDate.Day(), typedDate.Month(), typedDate.Year())
			},
		},
		YAxis: chart.YAxis{
			Style: chart.Style{
				Show: true,
			},
		},
	}

	graph.Elements = []chart.Renderable{chart.LegendThin(&graph)}

	s.saveToFile(graph, chartName)
}

func (s *ChartTradingStrategyService) collectTransactionsData(transactionProfitPercentsDtos []transaction.TransactionProfitPercentsDto) ([]time.Time, []float64) {
	var xvalues []time.Time
	var yvalues []float64

	for _, dto := range transactionProfitPercentsDtos {
		//start2023, _ := time.Parse(constants.DATE_FORMAT, "2023-01-01")
		//if dto.CreatedAt.After(start2023) {
		//	return xvalues, yvalues
		//}

		xvalues = append(xvalues, dto.CreatedAt)
		yvalues = append(yvalues, dto.ProfitPercent)
	}

	return xvalues, yvalues
}

func (s *ChartTradingStrategyService) collectWalletData(transactionProfitPercentsDtos []transaction.TransactionProfitPercentsDto) ([]time.Time, []float64) {
	var xvalues []time.Time
	var yvalues []float64

	walletCost := float64(s.InitialWalletCost)

	for _, dto := range transactionProfitPercentsDtos {
		//start2023, _ := time.Parse(constants.DATE_FORMAT, "2023-01-01")
		//if dto.CreatedAt.After(start2023) {
		//	return xvalues, yvalues
		//}

		walletCost += moneyUtil.CalculatePercentOf(float64(walletCost), dto.ProfitPercent/2)

		xvalues = append(xvalues, dto.CreatedAt)
		yvalues = append(yvalues, moneyUtil.GetDollarsByCents(int64(walletCost)))
	}

	return xvalues, yvalues
}

func (s *ChartTradingStrategyService) saveToFile(graph chart.Chart, fileName string) {
	buffer := bytes.NewBuffer([]byte{})
	err := graph.Render(chart.PNG, buffer)

	if err != nil {
		fmt.Println(err)
	}

	ioutil.WriteFile("charts/"+fileName+".png", buffer.Bytes(), 0644)
}
