package telegram

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/jamesread/japella/internal/runtimeconfig"
	log "github.com/sirupsen/logrus"
	pb "github.com/jamesread/japella/gen/protobuf"
	"github.com/jamesread/japella/internal/amqp"
	"strconv"

	"time"
)

var bot *tgbotapi.BotAPI

var cfg struct {
	Common   *runtimeconfig.CommonConfig
	Telegram struct {
		BotToken string
	}
}

type TelegramAdaptor struct {
}

func (n TelegramAdaptor) Start() {
	log.Infof("japella-adaptor-telegram")

	cfg.Common = &runtimeconfig.CommonConfig{}

	runtimeconfig.LoadConfigCommon(cfg.Common)
	runtimeconfig.LoadConfig("config.telegram.yaml", &cfg.Telegram)

	log.Infof("cfg: %+v", cfg)

	StartBot(cfg.Telegram.BotToken)

	for {
		time.Sleep(1 * time.Second)
	}
}

func StartBot(botToken string) {
	log.Infof("botToken: %v", botToken)

	var err error

	bot, err = tgbotapi.NewBotAPI(botToken)

	if err != nil {
		log.Panic(err)
	}

	bot.Debug = true

	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	go Replier()

	updates := bot.GetUpdatesChan(u)

	log.Infof("updates: %v", updates)

	for update := range updates {
		log.Infof("update: %v", update)

		if update.Message != nil { // If we got a message
			log.Infof("msg from [%s] %s", update.Message.From.UserName, update.Message.Text)

			amqp.PublishPb(&pb.IncommingMessage {
				Author: update.Message.From.UserName,
				Content: update.Message.Text,
				Channel: strconv.FormatInt(update.Message.Chat.ID, 10),
				Protocol: "telegram",
				Timestamp: time.Now().Unix(),
			})
		}
	}
}

func Replier() {
	amqp.ConsumeForever("telegram-OutgoingMessage", func(d amqp.Delivery) {
		reply := pb.OutgoingMessage{}

		amqp.Decode(d.Message.Body, &reply)

		log.Infof("reply: %v", &reply)

		channelId, _ := strconv.ParseInt(reply.Channel, 10, 64)
		msg := tgbotapi.NewMessage(channelId, reply.Content)

		messageId, _ := strconv.Atoi(reply.IncommingMessageId)

		log.Infof("messageId: %v %v", messageId, bot)
		msg.ReplyToMessageID = messageId

		bot.Send(msg)
	})
}
