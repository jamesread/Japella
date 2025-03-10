package exec

import (
	"bytes"
	"context"
	"fmt"
	"github.com/google/shlex"
	msgs "github.com/jamesread/japella/gen/japella/nodemsgs/v1"
	"github.com/jamesread/japella/internal/botbase"
	"github.com/jamesread/japella/internal/nanoservice"
	log "github.com/sirupsen/logrus"
	"os"
	"os/exec"
	"strings"
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

	bot.RegisterBangCommand("exechelp", bot.exechelp)
	bot.RegisterBangCommand("execrefresh", bot.refreshCommands)

	bot.refreshCommands(nil, "execrefresh", "")

	bot.ConsumeBangCommands().Wait()
}

func (b *ExecBot) refreshCommands(m *msgs.IncomingMessage, command string, arguments string) {
	files, err := os.ReadDir("/usr/libexec/japella/")

	if err != nil {
		b.Logger().Errorf("Error reading directory: %v", err)
		return
	}

	count := 0

	for _, file := range files {
		b.Logger().Infof("Registering bang command: %v", file.Name())
		b.RegisterBangCommand(file.Name(), b.execreq)

		commands += file.Name() + " "

		count++
	}

	if m != nil {
		reply := b.Reply(m)
		reply.Content = fmt.Sprintf("%v commands registered.", count)

		b.SendMessage(reply)
	}
}

func (b *ExecBot) exechelp(m *msgs.IncomingMessage, command string, arguments string) {
	out := &msgs.OutgoingMessage{
		Channel:  m.Channel,
		Protocol: m.Protocol,
	}

	out.Content = "Available commands: " + commands
	b.SendMessage(out)
}

func getCommandContext(ctx context.Context, command string) (*exec.Cmd, error) {
	args, err := shlex.Split(command)

	if err != nil {
		return nil, err
	}

	cmd := exec.CommandContext(ctx, "/usr/libexec/japella/"+args[0], args[1:]...)

	return cmd, nil
}

func (b *ExecBot) execreq(m *msgs.IncomingMessage, command string, arguments string) {
	out := &msgs.OutgoingMessage{
		Channel:  m.Channel,
		Protocol: m.Protocol,
	}

	b.Logger().Infof("Executing command: %v", m.Content)

	script := m.Content
	script = strings.ReplaceAll(script, ".", "")
	script = strings.ReplaceAll(script, "!", "")

	b.Logger().Infof("Executing command: %v", script)

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(10*time.Second))

	defer cancel()

	cmd, err := getCommandContext(ctx, script)

	var stderr bytes.Buffer
	var output string

	if err == nil {
		args := make(map[string]string)
		args["channel"] = m.Channel
		args["protocol"] = m.Protocol
		args["author"] = m.Author
		args["server"] = m.Server
		args["content"] = m.Content
		args["args"] = arguments
		cmd.Env = buildEnv(b.Logger(), args)
		cmd.Stderr = &stderr

		var outputBytes []byte

		outputBytes, err = cmd.Output()
		output = string(outputBytes)
	}

	if err != nil {
		b.Logger().Errorf("Error executing command: %v %v", err, stderr.String())

		out.Content = "Error executing command"
	} else {
		out.Content = output
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

		env = append(env, keyName+"="+value)
	}

	return env
}
