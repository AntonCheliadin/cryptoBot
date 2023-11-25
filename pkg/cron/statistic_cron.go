package cron

import (
	"cryptoBot/pkg/api/telegram"
	"cryptoBot/pkg/service/statistic"
	"github.com/go-co-op/gocron"
	"go.uber.org/zap"
	"time"
)

type statisticJob struct {
	service statistic.IStatisticService
}

func NewStatisticJob(service statistic.IStatisticService) *statisticJob {
	job := statisticJob{service: service}
	job.initStatisticJob()
	return &job
}

func (j *statisticJob) initStatisticJob() {
	s := gocron.NewScheduler(time.UTC)

	_, err := s.Cron("0 6 * * *").Do(j.execute) //every day at 6 (8 UA)
	if err != nil {
		zap.S().Errorf("Error during trading job %s", err.Error())
	}

	_, err2 := s.Cron("30 6-20 * * *").Do(j.executeHour) //every hour from 8 through 22 at 30min
	if err2 != nil {
		zap.S().Errorf("Error during trading job %s", err.Error())
	}

	s.StartAsync()
}

func (j *statisticJob) execute() {
	statistics := j.service.BuildStatistics()

	telegram.SendTextToTelegramChat(statistics)
}

func (j *statisticJob) executeHour() {
	statistics := j.service.BuildHourStatistics()

	telegram.SendTextToTelegramChat(statistics)
}
