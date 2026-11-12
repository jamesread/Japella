package exec

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/shlex"
	msgs "github.com/jamesread/japella/gen/japella/nodemsgs/v1"
	"github.com/jamesread/japella/internal/botbase"
	"github.com/jamesread/japella/internal/db"
	"github.com/jamesread/japella/internal/nanoservice"
	"github.com/jamesread/japella/internal/runtimeconfig"
)

type Exec struct {
	nanoservice.Nanoservice
}

type ExecBot struct {
	botbase.Bot
}

var commands = ""

func (e *Exec) Start() {
	bot := ExecBot{}
	bot.SetName("Exec")
	bot.Setup()

	if !runtimeconfig.Get().Amqp.Enabled {
		bot.Logger().Errorf("AMQP is not enabled, cannot start Exec bot.")
		return
	}

	bot.RegisterBangCommand("exechelp", bot.exechelp)
	bot.RegisterBangCommand("execrefresh", bot.refreshCommands)

	bot.refreshCommands(nil, "execrefresh", "")

	bot.ConsumeBangCommands().Wait()
}

func (b *ExecBot) refreshCommands(m *msgs.IncomingMessage, command string, arguments string) {
	files, err := os.ReadDir(scriptDir)

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
		Channel:         m.Channel,
		Protocol:        m.Protocol,
		Identity:        m.Identity, // Include bot identity to route to correct bot instance
		ConversationKey: db.BuildConversationKey(m.Channel, m.Author),
	}

	out.Content = "Available commands: " + commands
	b.SendMessage(out)
}

const scriptDir = "/usr/libexec/japella"

// command is the bang command that matched a registered script, so only its
// base name is used and the script always resolves inside scriptDir. Dots and
// "!" have always been stripped from arguments; that is kept for compatibility.
func getCommandContext(ctx context.Context, command string, arguments string) (*exec.Cmd, error) {
	args, err := shlex.Split(strings.NewReplacer(".", "", "!", "").Replace(arguments))
	if err != nil {
		return nil, err
	}

	script := filepath.Join(scriptDir, filepath.Base(command))

	return exec.CommandContext(ctx, script, args...), nil //nolint:gosec // script is confined to scriptDir; arguments are passed without a shell
}

func (b *ExecBot) execreq(m *msgs.IncomingMessage, command string, arguments string) {
	out := &msgs.OutgoingMessage{
		Channel:         m.Channel,
		Protocol:        m.Protocol,
		Identity:        m.Identity, // Include bot identity to route to correct bot instance
		ConversationKey: db.BuildConversationKey(m.Channel, m.Author),
	}

	b.Logger().Infof("Executing command: %v %v", command, arguments)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

	defer cancel()

	cmd, err := getCommandContext(ctx, command, arguments)

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
		cmd.Env = buildEnv(args)
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

func buildEnv(args map[string]string) []string {
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
