package support

import (
	"github.com/jamesread/japella/internal/botbase"
)

type SupportBot struct {
	botbase.Bot
}

func main() {
	bot := &SupportBot{}
	bot.Setup()
}
