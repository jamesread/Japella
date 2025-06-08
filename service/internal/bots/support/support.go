package support

import (
//	pb "github.com/jamesread/japella/gen/protobuf"
	"github.com/jamesread/japella/internal/botbase"
//	log "github.com/sirupsen/logrus"
)

type SupportBot struct {
	botbase.Bot
}

func main() {
	bot := &SupportBot{}
	bot.Setup()
	/*
	bot.RegisterBangCommand("threadsearch", bot.searchForThreads)

	go botbase.Consume(handleThreadSearchResponse)

	bot.ConsumeBangCommands().Wait()
	*/
}

/**
func (bot *SupportBot) searchForThreads(msg *pb.IncomingMessage) {
	bot.Logger().Infof("Searching for threads")

	res := &pb.ThreadSearchRequest{
		Protocol: "discord",
	}

	bot.Publish(res)
}

func handleThreadSearchResponse[M pb.ThreadSearchResponse](msg M) {
	log.Infof("Search result: %+v", &msg)
}
*/
