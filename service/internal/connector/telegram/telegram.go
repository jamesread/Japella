package telegram

import (
	tgbotapi "github.com/go-telegram/bot"
	tgbotmdl "github.com/go-telegram/bot/models"

	msgs "github.com/jamesread/japella/gen/japella/nodemsgs/v1"
	"github.com/jamesread/japella/internal/amqp"
	"github.com/jamesread/japella/internal/connector"
	"github.com/jamesread/japella/internal/runtimeconfig"
	"github.com/jamesread/japella/internal/utils"
	"strconv"

	"context"
	"os"
	"os/signal"

	"time"

	log "github.com/sirupsen/logrus"
)

type TelegramConnector struct {
	nickname string
	utils.LogComponent

	connector.BaseConnector
}

func (c *TelegramConnector) GetIdentity() string {
	return c.nickname
}

func (c *TelegramConnector) GetProtocol() string {
	return "telegram"
}

func (c *TelegramConnector) GetIcon() string {
	return "telegram"
}

func (c *TelegramConnector) StartWithConfig(startup *connector.ControllerStartupConfiguration) {
	config, _ := startup.Config.(*runtimeconfig.TelegramConfig)

	if config == nil || config.Token == "" {
		c.Logger().Errorf("Telegram bot token is not set in configuration")
		return
	}

	c.StartWithToken(config.Token)
}

func (c *TelegramConnector) StartWithToken(token string) {
	c.SetPrefix("Telegram-new")
	c.Logger().Infof("japella-bot-telegram")

	c.startBot(token)
}

var bot *tgbotapi.Bot

func (c *TelegramConnector) startBot(botToken string) {
	c.Logger().Infof("botToken: %v", botToken)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	opts := []tgbotapi.Option{
		tgbotapi.WithDefaultHandler(c.messageHandler),
	}

	var err error

	bot, err = tgbotapi.New(botToken, opts...)

	if err != nil {
		log.Errorf("Error creating bot: %v", err)
	}

	if runtimeconfig.Get().Amqp.Enabled {
		go c.Replier()
	}

	me, _ := bot.GetMe(ctx)
	c.nickname = me.Username

	c.SetPrefix("Telegram-" + c.nickname)
	c.Logger().Infof("Telegram getMe(): %+v", me)

	go bot.Start(ctx)
}

func (c TelegramConnector) messageHandler(ctx context.Context, b *tgbotapi.Bot, update *tgbotmdl.Update) {
	c.Logger().Infof("Telegram - update: %+v", update)

	if runtimeconfig.Get().Amqp.Enabled {
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
