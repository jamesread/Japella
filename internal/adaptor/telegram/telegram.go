package telegram

import (
	tgbotapi "github.com/go-telegram/bot"
	tgbotmdl "github.com/go-telegram/bot/models"

	"github.com/jamesread/japella/internal/runtimeconfig"
	log "github.com/sirupsen/logrus"
	pb "github.com/jamesread/japella/gen/protobuf"
	"github.com/jamesread/japella/internal/amqp"
	"github.com/jamesread/japella/internal/nanoservice"
	"github.com/go-kod/kod"
	"strconv"

	"os/signal"
	"os"
	"context"

	"time"
)

var cfg struct {
	Common   *runtimeconfig.CommonConfig
	Telegram struct {
		BotToken string
	}
}

type TelegramAdaptor struct {
	kod.Implements[nanoservice.Nanoservice]
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

var bot *tgbotapi.Bot

func StartBot(botToken string) {
	log.Infof("botToken: %v", botToken)

	var err error

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	if err != nil {
		log.Panic(err)
	}

	opts := []tgbotapi.Option {
		tgbotapi.WithDefaultHandler(messageHandler),
	}

	bot, err = tgbotapi.New(botToken, opts...)

	go Replier()

	me, _ := bot.GetMe(ctx)

	log.Infof("Telegram getMe(): %+v", me)

	bot.Start(ctx)
}

func messageHandler(ctx context.Context, b *tgbotapi.Bot, update *tgbotmdl.Update) {
	log.Infof("Telegram - update: %+v", update)

	if update.Message != nil { // If we got a message
		log.WithFields(log.Fields{
			"from": update.Message.From,
			"content": update.Message.Text,
		}).Infof("Telegram - message recevied");

		amqp.PublishPb(&pb.IncomingMessage {
//			Author: update.Message.From,
			Content: update.Message.Text,
			Channel: strconv.FormatInt(update.Message.Chat.ID, 10),
			Protocol: "telegram",
			Timestamp: time.Now().Unix(),
		})
	}
}

func Replier() {
	ctx := context.Background()

	amqp.ConsumeForever("telegram-OutgoingMessage", func(d amqp.Delivery) {
		reply := pb.OutgoingMessage{}

		amqp.Decode(d.Message.Body, &reply)

		log.Infof("reply: %v", &reply)

		channelId, _ := strconv.ParseInt(reply.Channel, 10, 64)
		bot.SendMessage(ctx, &tgbotapi.SendMessageParams{
			ChatID: channelId,
			Text: reply.Content,
		})

//		messageId, _ := strconv.Atoi(reply.IncomingMessageId)

//		log.Infof("messageId: %v %v", messageId, bot)
//		msg.ReplyToMessageID = messageId

//		bot.Send(msg)
	})
}
