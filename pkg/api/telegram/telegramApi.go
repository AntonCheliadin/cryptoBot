package telegram

import (
	"github.com/spf13/viper"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
)

func SendTextToTelegramChat(text string) {
	if !viper.GetBool("telegram.enabled") {
		//zap.S().Infof("Telegram: %s", text)
		return
	}
	var TELEGRAM_API = "https://api.telegram.org/bot" + os.Getenv("TELEGRAM_BOT_API_KEY") + "/sendMessage"
	var CHAT_ID = os.Getenv("TELEGRAM_BOT_CHAT_ID")

	response, err := http.PostForm(
		TELEGRAM_API,
		url.Values{
			"chat_id":    {CHAT_ID},
			"text":       {text},
			"parse_mode": {"HTML"},
		})

	if err != nil {
		log.Printf("error when posting text to the chat: %s", err.Error())
	}
	defer response.Body.Close()

	var _, errRead = ioutil.ReadAll(response.Body)
	if errRead != nil {
		log.Printf("error in parsing telegram answer %s", errRead.Error())
	}
}
