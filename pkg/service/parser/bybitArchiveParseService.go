package parser

import (
	"compress/gzip"
	"cryptoBot/pkg/constants"
	"cryptoBot/pkg/data/domains"
	"cryptoBot/pkg/repository"
	"cryptoBot/pkg/util"
	"encoding/csv"
	"errors"
	"fmt"
	"go.uber.org/zap"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

var bybitArchiveParseServiceImpl *BybitArchiveParseService

func NewBybitArchiveParseService(klineRepo repository.Kline) *BybitArchiveParseService {
	if bybitArchiveParseServiceImpl != nil {
		panic("Unexpected try to create second service instance")
	}
	bybitArchiveParseServiceImpl = &BybitArchiveParseService{
		klineRepo: klineRepo,
	}
	return bybitArchiveParseServiceImpl
}

type BybitArchiveParseService struct {
	klineRepo repository.Kline
}

func (s *BybitArchiveParseService) Parse(coin *domains.Coin, timeFrom time.Time, timeTo time.Time, intervalInMinutes int) error {
	timeIter := timeFrom
	for timeIter.Before(timeTo) {

		fileName := fmt.Sprintf("archive/%s%s.csv", coin.Symbol, timeIter.Format(constants.DATE_FORMAT))
		if _, err := os.Stat(fileName); errors.Is(err, os.ErrNotExist) {
			zap.S().Infof("File doesn't exist %s", fileName)

			if unzipErr := s.download(coin, timeIter); unzipErr != nil {
				return unzipErr
			}

			if unzipErr := s.unzip(coin, timeIter); unzipErr != nil {
				return unzipErr
			}
		}

		data, err := s.parseFile(coin, timeIter)
		if err != nil {
			return err
		}

		err = s.parseData(coin, timeIter, intervalInMinutes, data)
		if err != nil {
			return err
		}

		timeIter = timeIter.Add(time.Hour * 24)
	}

	return nil
}

func (s *BybitArchiveParseService) download(coin *domains.Coin, day time.Time) error {
	fullURLFile := fmt.Sprintf("https://public.bybit.com/trading/%s/%s%s.csv.gz", coin.Symbol, coin.Symbol, day.Format(constants.DATE_FORMAT))

	// Build fileName from fullPath
	fileURL, err := url.Parse(fullURLFile)
	if err != nil {
		return err
	}
	path := fileURL.Path
	segments := strings.Split(path, "/")
	fileName := segments[len(segments)-1]

	zap.S().Infof("Download %s", fileName)

	// Create blank file
	file, err := os.Create("archive/" + fileName)
	if err != nil {
		return err
	}
	client := http.Client{
		CheckRedirect: func(r *http.Request, via []*http.Request) error {
			r.URL.Opaque = r.URL.Path
			return nil
		},
	}
	// Put content on file
	resp, err := client.Get(fullURLFile)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	size, err := io.Copy(file, resp.Body)

	defer file.Close()

	zap.S().Infof("Downloaded a file %s with size %d", fileName, size)
	return nil
}

func (s *BybitArchiveParseService) unzip(coin *domains.Coin, day time.Time) error {
	fileName := fmt.Sprintf("archive/%s%s.csv", coin.Symbol, day.Format(constants.DATE_FORMAT))
	zap.S().Infof("Unzip %s", fileName)

	gzipfile, err := os.Open(fileName + ".gz")

	if err != nil {
		return err
	}

	reader, err := gzip.NewReader(gzipfile)
	if err != nil {
		return err
	}
	defer reader.Close()

	writer, err := os.Create(fileName)

	if err != nil {
		return err
	}

	defer writer.Close()

	if _, err = io.Copy(writer, reader); err != nil {
		return err
	}

	return nil
}

func (s *BybitArchiveParseService) parseFile(coin *domains.Coin, day time.Time) ([][]string, error) {
	fileName := fmt.Sprintf("archive/%s%s.csv", coin.Symbol, day.Format(constants.DATE_FORMAT))
	zap.S().Infof("Parse %s", fileName)
	f, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}

	defer f.Close()

	csvReader := csv.NewReader(f)
	data, err := csvReader.ReadAll()
	if err != nil {
		return nil, err
	}

	return data, nil
}

func (s *BybitArchiveParseService) parseData(coin *domains.Coin, day time.Time, intervalInMinutes int, data [][]string) error {
	var kline *domains.Kline

	i := 1
	sign := 1

	firstTimestampInSeconds, _ := strconv.Atoi(strings.Split(data[1][0], ".")[0])
	lastTimestampInSeconds, _ := strconv.Atoi(strings.Split(data[len(data)-1][0], ".")[0])

	if lastTimestampInSeconds < firstTimestampInSeconds {
		i = len(data) - 1
		sign = -1
	}

	for ; i > 0 && i < len(data); i += sign {
		line := data[i]
		segments := strings.Split(line[0], ".")
		timestampInSeconds, _ := strconv.Atoi(segments[0])
		tickTime := util.GetTimeBySeconds(timestampInSeconds).UTC()
		klineOpenTime := util.RoundToMinutesWithInterval(tickTime, strconv.Itoa(intervalInMinutes))

		price, _ := strconv.ParseFloat(line[4], 64)
		priceInCents := util.GetCents(price)

		if kline != nil && util.InTimeSpanInclusive(kline.OpenTime, kline.CloseTime, tickTime) {
			kline.Close = priceInCents
			if kline.High < priceInCents {
				kline.High = priceInCents
			}
			if kline.Low > priceInCents {
				kline.Low = priceInCents
			}
		} else {
			if kline != nil {
				s.klineRepo.SaveKline(kline)
			}
			kline = s.findOrCreateKline(coin, klineOpenTime, tickTime, intervalInMinutes, priceInCents)
		}
	}

	if kline != nil {
		s.klineRepo.SaveKline(kline)
	}
	return nil
}

func (s *BybitArchiveParseService) findOrCreateKline(coin *domains.Coin, klineOpenTime, tickTime time.Time, intervalInMinutes int, priceInCents int64) *domains.Kline {
	kline, _ := s.klineRepo.FindOpenedAtMoment(coin.Id, klineOpenTime, strconv.Itoa(intervalInMinutes))
	if kline != nil {
		return kline
	}

	return &domains.Kline{
		CoinId:    coin.Id,
		OpenTime:  klineOpenTime,
		CloseTime: klineOpenTime.Add(time.Minute * time.Duration(intervalInMinutes)),
		Interval:  strconv.Itoa(intervalInMinutes),
		Open:      priceInCents,
		Close:     priceInCents,
		High:      priceInCents,
		Low:       priceInCents,
	}
}
