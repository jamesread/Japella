package exec

import (
	"github.com/jamesread/japella/internal/nanoservice"
	"github.com/jamesread/japella/internal/botbase"
	pb "github.com/jamesread/japella/gen/protobuf"
	log "github.com/sirupsen/logrus"
	"os"
	"os/exec"
	"strings"
	"fmt"
	"context"
	"time"
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

func(b *ExecBot) exechelp(m *pb.IncomingMessage, command string, arguments string) {
	out := &pb.OutgoingMessage{
		Channel: m.Channel,
		Protocol: m.Protocol,
	}

	out.Content = "Available commands: " + commands
	b.SendMessage(out)
}

func(b *ExecBot) execreq(m *pb.IncomingMessage, command string, arguments string) {
	out := &pb.OutgoingMessage{
		Channel: m.Channel,
		Protocol: m.Protocol,
	}

	b.Logger().Infof("Executing command: %v", m.Content)

	script := m.Content
	script = strings.ReplaceAll(script, ".", "")
	script = strings.ReplaceAll(script, "!", "")

	b.Logger().Infof("Executing command: %v", script)

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(10*time.Second))
	cmd := exec.CommandContext(ctx, "sh", "-c", "/usr/libexec/japella/" + script)

	defer cancel()

	args := make(map[string]string)
	args["channel"] = m.Channel
	args["protocol"] = m.Protocol
	args["author"] = m.Author
	args["server"] = m.Server
	args["content"] = m.Content
	args["args"] = arguments
	cmd.Env = buildEnv(b.Logger(), args)

	output, err := cmd.Output()

	if err != nil {
		b.Logger().Errorf("Error executing command: %v", err)
		out.Content = "Error executing command"
	} else {
		out.Content = string(output)
	}

	b.SendMessage(out)
}

func buildEnv(logger *log.Logger, args map[string]string) []string {
	env := os.Environ()
	env = append(env, "JAPELLA=1")

	for key, value := range args {
		keyName := fmt.Sprintf("%v", strings.TrimSpace(strings.ToUpper(key)))

		if keyName == "" {
			continue
		}

		env = append(env, keyName + "=" + value)
	}

	return env
}
