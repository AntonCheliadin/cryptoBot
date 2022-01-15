package telegram

import (
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
)

func SendTextToTelegramChat(text string) {
	var TELEGRAM_API = "https://api.telegram.org/bot" + os.Getenv("TELEGRAM_BOT_API_KEY") + "/sendMessage"
	var CHAT_ID = os.Getenv("TELEGRAM_BOT_CHAT_ID")

	log.Printf("Sending [%s]", text)
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

	var bodyBytes, errRead = ioutil.ReadAll(response.Body)
	if errRead != nil {
		log.Printf("error in parsing telegram answer %s", errRead.Error())
	}
	bodyString := string(bodyBytes)
	log.Printf("Body of Telegram Response: %s", bodyString)
}
