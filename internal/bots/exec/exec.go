package exec

import (
	"github.com/jamesread/japella/internal/nanoservice"
	"github.com/jamesread/japella/internal/botbase"
	pb "github.com/jamesread/japella/gen/protobuf"
	"os/exec"
)

type Exec struct {
	nanoservice.Nanoservice
}

type ExecBot struct {
	botbase.Bot
}

func (e Exec) Start() {
	bot := ExecBot{}
	bot.SetName("Exec")
	bot.Setup()
	bot.RegisterBangCommand("execreq", bot.execreq)
	bot.RegisterBangCommand("nextweekend", bot.nextweekend)
	bot.ConsumeBangCommands().Wait()
}

func (b *ExecBot) execreq(m *pb.IncommingMessage) {
	out := &pb.OutgoingMessage{
		Channel: m.Channel,
		Content: "execreq",
		Protocol: "telegram",
	}

	b.Logger().Info("execreq")

	b.SendMessage(out)
}

func(b *ExecBot) nextweekend(m *pb.IncommingMessage) {
	out := &pb.OutgoingMessage{
		Channel: m.Channel,
		Content: "nextweekend",
		Protocol: "telegram",
	}

	// exec nextweekend.py and get output

	cmd := exec.Command("nextweekend.py")
	output, err := cmd.Output()

	if err != nil {
		out.Content = "Error executing command"
	} else {
		out.Content = string(output)
	}

	b.SendMessage(out)
}
