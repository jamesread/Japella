package main

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/jamesread/japella/internal/runtimeconfig"
	log "github.com/sirupsen/logrus"

	"time"
)

var cfg struct {
	Common   *runtimeconfig.CommonConfig
	Telegram struct {
		BotToken string
	}
}

func main() {
	log.Infof("japella-adaptor-telegram")

	runtimeconfig.LoadConfigCommon(cfg.Common)
	runtimeconfig.LoadConfig("config.telegram.yaml", cfg.Telegram)

	log.Infof("cfg: %+v", cfg)

	Start(cfg.Telegram.BotToken)

	for {
		time.Sleep(1 * time.Second)
	}
}

func Start(botToken string) {
	log.Infof("botToken: %v", botToken)

	bot, err := tgbotapi.NewBotAPI(botToken)

	if err != nil {
		log.Panic(err)
	}

	bot.Debug = true

	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message != nil { // If we got a message
			log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)

			msg := tgbotapi.NewMessage(update.Message.Chat.ID, update.Message.Text)
			msg.ReplyToMessageID = update.Message.MessageID

			bot.Send(msg)
		}
	}
}
