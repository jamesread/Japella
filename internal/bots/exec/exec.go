package exec

import (
	"github.com/jamesread/japella/internal/nanoservice"
	"github.com/jamesread/japella/internal/botbase"
	pb "github.com/jamesread/japella/gen/protobuf"
	"os"
	"os/exec"
	"strings"
)

type Exec struct {
	nanoservice.Nanoservice
}

type ExecBot struct {
	botbase.Bot
}

var commands = ""

func (e Exec) Start() {
	bot := ExecBot{}
	bot.SetName("Exec")
	bot.Setup()

	// List files in /usr/libexec/japella/

	files, err := os.ReadDir("/usr/libexec/japella/")

	if err != nil {
		bot.Logger().Errorf("Error reading directory: %v", err)
		return
	}

	for _, file := range files {
		bot.Logger().Infof("Registering bang command: %v", file.Name())
		bot.RegisterBangCommand(file.Name(), bot.execreq)

		commands += file.Name() + " "
	}

	bot.RegisterBangCommand("exechelp", bot.exechelp)

	bot.ConsumeBangCommands().Wait()
}

func(b *ExecBot) exechelp(m *pb.IncommingMessage) {
	out := &pb.OutgoingMessage{
		Channel: m.Channel,
		Protocol: "telegram",
	}

	out.Content = "Available commands: " + commands
	b.SendMessage(out)
}

func(b *ExecBot) execreq(m *pb.IncommingMessage) {
	out := &pb.OutgoingMessage{
		Channel: m.Channel,
		Protocol: "telegram",
	}

	b.Logger().Infof("Executing command: %v", m.Content)

	script := m.Content
	script = strings.ReplaceAll(script, ".", "")
	script = strings.ReplaceAll(script, "!", "")

	b.Logger().Infof("Executing command: %v", script)

	cmd := exec.Command("sh", "-c", "/usr/libexec/japella/" + script)
	output, err := cmd.Output()

	if err != nil {
		b.Logger().Errorf("Error executing command: %v", err)
		out.Content = "Error executing command"
	} else {
		out.Content = string(output)
	}

	b.SendMessage(out)
}
