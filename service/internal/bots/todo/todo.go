package todo

import (
	pb "github.com/jamesread/japella/gen/japella/nodemsgs/v1"
	"github.com/jamesread/japella/internal/botbase"
)

type TodoBotImpl struct {
	botbase.Bot

	list []string
}

func (bot *TodoBotImpl) Name() string {
	return "Todo"
}

func (bot *TodoBotImpl) Start() {
	bot.SetName("TodoBotImpl")
	bot.Logger().Infof("Starting %v bot", bot.Name())

	bot.list = make([]string, 0)

	bot.Setup()
	bot.RegisterBangCommand("todo", bot.onNew)
	bot.RegisterBangCommand("done", bot.onDone)

	bot.ConsumeBangCommands().Wait()

	bot.Logger().Infof("Exiting")
}

func (bot *TodoBotImpl) onNew(msg *pb.IncomingMessage, a string, b string) {
	/**
	bot.Logger().Infof("Creating new todo")

	bot.list = append(bot.list, msg.Content)
	*/
}

func (bot *TodoBotImpl) onDone(msg *pb.IncomingMessage, a string, b string) {
	/*
		bot.Logger().Infof("Completing todo")

		bot.list = bot.list[1:]
	*/
}
