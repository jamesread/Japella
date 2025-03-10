package telegram

import (
	tgbotapi "github.com/go-telegram/bot"
	tgbotmdl "github.com/go-telegram/bot/models"

	msgs "github.com/jamesread/japella/gen/japella/nodemsgs/v1"
	"github.com/jamesread/japella/internal/amqp"
	"github.com/jamesread/japella/internal/nanoservice"
	"github.com/jamesread/japella/internal/runtimeconfig"
	"github.com/jamesread/japella/internal/utils"
	"strconv"

	"context"
	"os"
	"os/signal"

	"time"
)

type TelegramConnector struct {
	nanoservice.Nanoservice
	utils.LogComponent
}

func (c TelegramConnector) Start() {
	c.SetPrefix("Telegram")
	c.Logger().Infof("Telegram connector started")

	cfg := runtimeconfig.Get()

	c.startBot(cfg.Connectors.Telegram.BotToken)

	for {
		time.Sleep(1 * time.Second)
	}
}

var bot *tgbotapi.Bot

func (c TelegramConnector) startBot(botToken string) {
	c.Logger().Infof("botToken: %v", botToken)

	var err error

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	if err != nil {
		c.Logger().Panic(err)
	}

	opts := []tgbotapi.Option{
		tgbotapi.WithDefaultHandler(c.messageHandler),
	}

	bot, err = tgbotapi.New(botToken, opts...)

	go c.Replier()

	me, _ := bot.GetMe(ctx)

	c.Logger().Infof("Telegram getMe(): %+v", me)

	bot.Start(ctx)
}

func (c TelegramConnector) messageHandler(ctx context.Context, b *tgbotapi.Bot, update *tgbotmdl.Update) {
	c.Logger().Infof("Telegram - update: %+v", update)

	if update.Message != nil { // If we got a message
		c.Logger().Infof("Telegram - message recevied from:%v content:%v", update.Message.From, update.Message.Text)

		amqp.PublishPb(&msgs.IncomingMessage{
			Author:    update.Message.From.Username,
			Content:   update.Message.Text,
			Channel:   strconv.FormatInt(update.Message.Chat.ID, 10),
			Protocol:  "telegram",
			Timestamp: time.Now().Unix(),
		})
	}
}

func (c TelegramConnector) Replier() {
	ctx := context.Background()

	amqp.ConsumeForever("telegram-OutgoingMessage", func(d amqp.Delivery) {
		reply := msgs.OutgoingMessage{}

		amqp.Decode(d.Message.Body, &reply)

		c.Logger().Infof("reply: %v", &reply)

		channelId, _ := strconv.ParseInt(reply.Channel, 10, 64)
		bot.SendMessage(ctx, &tgbotapi.SendMessageParams{
			ChatID: channelId,
			Text:   reply.Content,
		})

		//		messageId, _ := strconv.Atoi(reply.IncomingMessageId)

		//		log.Infof("messageId: %v %v", messageId, bot)
		//		msg.ReplyToMessageID = messageId

		//		bot.Send(msg)
	})
}
