package controller

import (
	telegramDto "cryptoBot/pkg/data/dto/telegram"
	"cryptoBot/pkg/service/telegram"
	"encoding/json"
	"github.com/go-chi/chi"
	"go.uber.org/zap"
	"io/ioutil"
	"log"
	"net/http"
)

func InitControllers(telegramService *telegram.TelegramService) *chi.Mux {
	r := chi.NewRouter()

	InitHealthCheckEndpoints(r)
	InitTelegramWebhookEndpoints(r, telegramService)
	return r
}

func InitHealthCheckEndpoints(r *chi.Mux) {
	r.Get("/healthcheck", func(res http.ResponseWriter, req *http.Request) {})
}

func InitTelegramWebhookEndpoints(r *chi.Mux, telegramService *telegram.TelegramService) {
	r.Post("/telegram/webhook", func(res http.ResponseWriter, req *http.Request) {

		var update, err = parseTelegramRequest(req)
		if err != nil {
			log.Printf("error parsing update, %s", err.Error())
			return
		}

		telegramService.HandleMessage(update)
	})
}

func parseTelegramRequest(r *http.Request) (*telegramDto.Update, error) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		zap.S().Errorf("API error: %s", err)
		return nil, err
	}
	zap.S().Infof("API response: %s", string(body))

	dto := telegramDto.Update{}
	errUnmarshal := json.Unmarshal(body, &dto)
	if errUnmarshal != nil {
		zap.S().Error("Unmarshal error", errUnmarshal.Error())
		return nil, errUnmarshal
	}

	return &dto, nil
}
