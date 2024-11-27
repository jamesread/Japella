package main

import (
	"github.com/jamesread/japella/internal/botbase"
	pb "github.com/jamesread/japella/gen/protobuf"
	log "github.com/sirupsen/logrus"
)

type SupportBot struct {
	botbase.Bot
}

func main() {
	bot := &SupportBot{}
	bot.Setup("support")
	bot.RegisterBangCommand("threadsearch", bot.searchForThreads)

	go botbase.Consume(handleThreadSearchResponse)

	bot.ConsumeBangCommands().Wait()
}

func (bot *SupportBot) searchForThreads(msg *pb.IncommingMessage) {
	bot.Logger().Infof("Searching for threads")

	res := &pb.ThreadSearchRequest{
		Protocol: "discord",
	}

	bot.Publish(res)
}

func handleThreadSearchResponse[M pb.ThreadSearchResponse](msg M) {
	log.Infof("Search result: %+v", &msg)
}
