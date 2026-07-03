package discord

import (
	"fmt"
	"sync"

	"github.com/jamesread/japella/internal/chatbot"
	"github.com/jamesread/japella/internal/connector"
	"github.com/jamesread/japella/internal/db"
	"github.com/jamesread/japella/internal/runtimeconfig"
	"github.com/jamesread/japella/internal/utils"
)

type DiscordConnector struct {
	botID         string
	nickname      string
	displayName   string
	isRunning     bool
	statusMessage string
	errorMessage  string
	botToken      string
	appID         string
	publicKey     string
	botMu         sync.Mutex
	hooks         []runtimeconfig.IncomingMessageHook
	db            *db.DB

	utils.LogComponent
	connector.BaseConnector
}

func (a *DiscordConnector) failStartup(statusMsg, errMsg string) {
	a.isRunning = false
	a.statusMessage = statusMsg
	a.errorMessage = errMsg
	connector.LogChatBotStartupFailure(a.db, a.GetProtocol(), a.nickname, errMsg)
}

func (a *DiscordConnector) SetStartupConfiguration(startup *connector.ControllerStartupConfiguration) {
	a.db = startup.DB
	cfg, _ := startup.Config.(*connector.ChatBotStartupConfig)
	if cfg == nil || cfg.BotID == "" {
		a.failStartup("Configuration error: Bot instance is not configured", "Chat bot instance configuration is missing")
		return
	}

	a.botID = cfg.BotID
	a.displayName = cfg.DisplayName
	a.botToken = a.db.GetCvarString(chatbot.DiscordTokenKey(cfg.BotID))
	a.appID = a.db.GetCvarString(chatbot.DiscordAppIDKey(cfg.BotID))
	a.publicKey = a.db.GetCvarString(chatbot.DiscordPublicKeyKey(cfg.BotID))
	if a.botToken == "" {
		a.failStartup("Configuration error: Bot token is not set", "Discord bot token is missing; set it when creating or editing the bot")
		return
	}

	a.isRunning = false
	a.statusMessage = "Stopped (not connected)"
	a.errorMessage = ""
}

func (a *DiscordConnector) Start() {
	// Bot connection is started explicitly via StartChatBot
}

func (a *DiscordConnector) StartChatBot() error {
	a.botMu.Lock()
	defer a.botMu.Unlock()

	if a.isRunning {
		return fmt.Errorf("discord bot is already running")
	}
	if a.botToken == "" {
		a.botToken = a.db.GetCvarString(chatbot.DiscordTokenKey(a.botID))
	}
	if a.botToken == "" {
		return fmt.Errorf("discord bot token is not configured")
	}

	a.SetPrefix("Discord")
	a.Logger().Infof("Discord connector starting")

	session := a.startActual(a.botToken)
	if session == nil {
		if a.errorMessage != "" {
			return fmt.Errorf("%s", a.errorMessage)
		}
		return fmt.Errorf("failed to start discord bot")
	}

	if runtimeconfig.Get().Amqp.Enabled {
		go a.Replier()
	}

	a.Logger().Infof("Discord connector started successfully")
	return nil
}

func (a *DiscordConnector) StopChatBot() error {
	a.botMu.Lock()
	defer a.botMu.Unlock()

	if !a.isRunning {
		a.statusMessage = "Stopped (not connected)"
		a.errorMessage = ""
		return nil
	}

	if goBot != nil {
		if err := goBot.Close(); err != nil {
			a.Logger().Warnf("Error closing Discord connection: %v", err)
		}
		goBot = nil
	}

	a.isRunning = false
	a.statusMessage = "Stopped (not connected)"
	a.errorMessage = ""
	a.Logger().Infof("Discord bot stopped")
	return nil
}

func (a *DiscordConnector) GetBotID() string {
	return a.botID
}

func (a *DiscordConnector) OnRefresh(socialAccount *db.SocialAccount) error {
	// Discord uses bot tokens, not OAuth, so no refresh is needed
	return nil
}
