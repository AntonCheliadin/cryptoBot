package snapshot

import (
	"bytes"
	"cryptoBot/pkg/constants"
	"cryptoBot/pkg/constants/futureType"
	"cryptoBot/pkg/data/domains"
	"cryptoBot/pkg/repository"
	"cryptoBot/pkg/service/indicator"
	"fmt"
	"github.com/blend/go-sdk/mathutil"
	"github.com/sdcoffey/big"
	"github.com/wcharczuk/go-chart"
	"github.com/wcharczuk/go-chart/drawing"
	"github.com/wcharczuk/go-chart/util"
	"io/ioutil"
	"strconv"
	"time"
)

var snapshotOrderServiceImpl *SnapshotOrderService

func NewSnapshotOrderService(transactionRepo repository.Transaction, klineRepo repository.Kline,
	localExtremumTrendService *indicator.LocalExtremumTrendService) *SnapshotOrderService {
	if snapshotOrderServiceImpl != nil {
		panic("Unexpected try to create second service instance")
	}
	snapshotOrderServiceImpl = &SnapshotOrderService{
		klineRepo:                 klineRepo,
		transactionRepo:           transactionRepo,
		LocalExtremumTrendService: localExtremumTrendService,
	}
	return snapshotOrderServiceImpl
}

type SnapshotOrderService struct {
	klineRepo                 repository.Kline
	transactionRepo           repository.Transaction
	LocalExtremumTrendService *indicator.LocalExtremumTrendService
}

func (s *SnapshotOrderService) SnapshotOrder(coin *domains.Coin, openTransaction *domains.Transaction, closeTransaction *domains.Transaction, interval string) {

	xvalues, yvalues := s.collectPriceData(coin, openTransaction, closeTransaction, interval)
	mainSeries := chart.TimeSeries{
		Name: coin.Symbol,
		Style: chart.Style{
			Show:        true,
			StrokeColor: chart.ColorBlue,
			FillColor:   chart.ColorBlue.WithAlpha(100),
		},
		XValues: xvalues,
		YValues: yvalues,
	}

	graph := chart.Chart{
		Width:  len(xvalues) * 30,
		Height: 500 + int((mathutil.Max(yvalues) - mathutil.Min(yvalues))),
		Background: chart.Style{
			Padding: chart.Box{
				Top: 50,
			},
		},
		YAxis: chart.YAxis{
			Name:      "Price",
			NameStyle: chart.StyleShow(),
			Style:     chart.StyleShow(),
			ValueFormatter: func(v interface{}) string {
				return fmt.Sprintf("%v$", int(v.(float64)))
			},
			GridLines: s.getPointsOfOpenCloseTime(openTransaction, closeTransaction),
		},
		XAxis: chart.XAxis{
			Style: chart.Style{
				Show: true,
			},
			ValueFormatter: chart.TimeHourValueFormatter,
			GridMajorStyle: chart.Style{
				Show:        true,
				StrokeColor: chart.ColorAlternateGray,
				StrokeWidth: 1.0,
			},
			GridLines: s.getPointsOfOpenClosePrice(openTransaction, closeTransaction),
		},
		Series: []chart.Series{
			mainSeries,
			s.getOrderLabels(openTransaction, closeTransaction),
		},
	}

	graph.Elements = []chart.Renderable{chart.LegendThin(&graph)}

	s.saveToFile(graph, s.buildFileName(openTransaction, closeTransaction))
}

func (s *SnapshotOrderService) buildEmaSeries(mainSeries chart.TimeSeries, period int, color drawing.Color) *chart.EMASeries {
	ma := &chart.EMASeries{
		Name: "EMA " + strconv.Itoa(period),
		Style: chart.Style{
			Show:            true,
			StrokeColor:     color,
			StrokeDashArray: []float64{5.0, 5.0},
		},
		Period:      period,
		InnerSeries: mainSeries,
	}
	return ma
}

func (s *SnapshotOrderService) findPrevExtremumKline(coin *domains.Coin, openTransaction *domains.Transaction, interval string) *domains.Kline {
	if openTransaction.FuturesType == futureType.SHORT {
		return s.LocalExtremumTrendService.FindNearestHighExtremum(coin, interval, openTransaction.CreatedAt)
	} else {
		return s.LocalExtremumTrendService.FindNearestLowExtremum(coin, interval, openTransaction.CreatedAt)
	}
}

func (s *SnapshotOrderService) collectPriceData(coin *domains.Coin, openTransaction *domains.Transaction, closeTransaction *domains.Transaction, interval string) ([]time.Time, []float64) {
	var xvalues []time.Time
	var yvalues []float64

	intervalMin, err := strconv.ParseInt(interval, 10, 64)
	extremumKline := s.findPrevExtremumKline(coin, openTransaction, interval)

	firstKlineTime := extremumKline.OpenTime.Add(time.Minute * time.Duration(-intervalMin*int64(100)))
	lastKlineTime := closeTransaction.CreatedAt.Add(time.Minute * time.Duration(intervalMin*int64(25)))

	klines, err := s.klineRepo.FindAllByCoinIdAndIntervalAndCloseTimeInRange(coin.Id, interval, firstKlineTime, lastKlineTime)
	if err != nil {
		fmt.Println(err.Error())
	}

	for _, kline := range klines {
		xvalues = append(xvalues, kline.CloseTime)
		yvalues = append(yvalues, kline.Close)
	}

	return xvalues, yvalues
}

func (s *SnapshotOrderService) getOrderLabels(openTransaction *domains.Transaction, closeTransaction *domains.Transaction) chart.AnnotationSeries {
	annotations := []chart.Value2{
		{
			XValue: util.Time.ToFloat64(closeTransaction.CreatedAt),
			YValue: (closeTransaction.Price),
			Label:  "Profit " + strconv.Itoa(int(closeTransaction.Profit.Int64)),
		},
	}

	if closeTransaction.Profit.Int64 > 0 {
		annotations = append(annotations, chart.Value2{
			XValue: util.Time.ToFloat64(openTransaction.CreatedAt),
			YValue: (openTransaction.StopLossPrice.Float64),
			Label:  "Stop loss",
		})
	} else {
		annotations = append(annotations, chart.Value2{
			XValue: util.Time.ToFloat64(openTransaction.CreatedAt),
			YValue: (openTransaction.TakeProfitPrice.Float64),
			Label:  "Take profit",
		})
	}

	return chart.AnnotationSeries{
		Annotations: annotations,
	}
}

func (s *SnapshotOrderService) getPointsOfOpenCloseTime(openTransaction *domains.Transaction, closeTransaction *domains.Transaction) []chart.GridLine {
	return []chart.GridLine{
		{Value: (openTransaction.Price)},
		{Value: (openTransaction.StopLossPrice.Float64)},
		{Value: (openTransaction.TakeProfitPrice.Float64)},
		{Value: (closeTransaction.Price)},
	}
}

func (s *SnapshotOrderService) getPointsOfOpenClosePrice(openTransaction *domains.Transaction, closeTransaction *domains.Transaction) []chart.GridLine {
	return []chart.GridLine{
		{Value: util.Time.ToFloat64(openTransaction.CreatedAt)},
		{Value: util.Time.ToFloat64(closeTransaction.CreatedAt)},
	}
}

func (s *SnapshotOrderService) buildFileName(transaction, closeTransaction *domains.Transaction) string {
	name := strconv.Itoa(int(transaction.Id)) + "-" +
		strconv.Itoa(int(transaction.TradingStrategy)) + "-" +
		futureType.GetString(transaction.FuturesType) + "-" +
		transaction.CreatedAt.Format(constants.DATE_FORMAT)

	if closeTransaction.Profit.Valid {
		name = name + fmt.Sprintf("-PROFIT[%v]", big.NewDecimal(closeTransaction.PercentProfit.Float64).FormattedString(2))
	}
	return name
}

func (s *SnapshotOrderService) saveToFile(graph chart.Chart, fileName string) {
	buffer := bytes.NewBuffer([]byte{})
	err := graph.Render(chart.PNG, buffer)

	if err != nil {
		fmt.Println(err)
	}

	ioutil.WriteFile("snapshots/"+fileName+".png", buffer.Bytes(), 0644)
}
