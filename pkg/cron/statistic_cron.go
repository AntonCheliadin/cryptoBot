package cron

import (
	"cryptoBot/pkg/api/telegram"
	"cryptoBot/pkg/service/statistic"
	"github.com/jasonlvhit/gocron"
	"go.uber.org/zap"
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
	err := gocron.Every(60 * 6).Minutes().Do(j.execute)
	if err != nil {
		zap.S().Errorf("Error during trading job %s", err.Error())
	}

	err2 := gocron.Every(60).Minutes().Do(j.executeHour)
	if err2 != nil {
		zap.S().Errorf("Error during trading job %s", err.Error())
	}
}

func (j *statisticJob) execute() {
	statistics := j.service.BuildStatistics()

	telegram.SendTextToTelegramChat(statistics)
}

func (j *statisticJob) executeHour() {
	statistics := j.service.BuildHourStatistics()

	telegram.SendTextToTelegramChat(statistics)
}
