package health

import (
	"context"
	"fmt"
	"sort"
	"strings"

	armhealth "github.com/jamesread/armature-health"
	"github.com/jamesread/japella/internal/connector"
)

// ConnectorConfigCheck surfaces configuration issues reported by started connectors.
type ConnectorConfigCheck struct {
	Connectors ConnectorSource
}

func (c ConnectorConfigCheck) ID() string    { return "connectors.configuration" }
func (c ConnectorConfigCheck) Title() string { return "Connector configuration issues" }

func (c ConnectorConfigCheck) Run(_ context.Context) ([]armhealth.Issue, error) {
	if c.Connectors == nil {
		return nil, nil
	}
	var issues []armhealth.Issue
	for _, id := range sortedConnectorIDs(c.Connectors.GetServices()) {
		svc := c.Connectors.GetServices()[id]
		issues = append(issues, connectorConfigIssues(id, svc)...)
	}
	return issues, nil
}

func connectorConfigIssues(id string, svc connector.BaseConnector) []armhealth.Issue {
	rawIssues := rawConnectorConfigIssues(svc)
	if len(rawIssues) == 0 {
		return nil
	}
	name := connectorDisplayName(svc)
	out := make([]armhealth.Issue, 0, len(rawIssues))
	for idx, issue := range rawIssues {
		out = append(out, armhealth.Issue{
			Key:      fmt.Sprintf("connector:%s:%d", id, idx),
			Severity: armhealth.SeverityWarning,
			Title:    fmt.Sprintf("Connector %q has a configuration issue", name),
			Detail:   issue.Message,
			Link:     connectorIssueLink(issue),
		})
	}
	return out
}

func rawConnectorConfigIssues(svc connector.BaseConnector) []connector.ConfigurationIssue {
	cfgProvider, ok := svc.(connector.ConfigProvider)
	if !ok || cfgProvider == nil {
		return nil
	}
	if result := cfgProvider.CheckConfiguration(); result != nil {
		return result.Issues
	}
	return nil
}

func connectorDisplayName(svc connector.BaseConnector) string {
	if dn, ok := svc.(connector.ConnectorWithDisplayName); ok {
		if name := strings.TrimSpace(dn.GetDisplayName()); name != "" {
			return name
		}
	}
	return svc.GetProtocol()
}

func connectorIssueLink(issue connector.ConfigurationIssue) string {
	if issue.FixPath == "" {
		return "/connectors"
	}
	if issue.FixHash != "" {
		return issue.FixPath + "#" + issue.FixHash
	}
	return issue.FixPath
}

// ChatBotsFailingCheck finds started chat bot instances that failed to start or have an error.
type ChatBotsFailingCheck struct {
	Connectors ConnectorSource
}

func (c ChatBotsFailingCheck) ID() string    { return "chatbots.failing" }
func (c ChatBotsFailingCheck) Title() string { return "Failing chat bots" }

func (c ChatBotsFailingCheck) Run(_ context.Context) ([]armhealth.Issue, error) {
	if c.Connectors == nil {
		return nil, nil
	}
	var issues []armhealth.Issue
	for _, id := range sortedConnectorIDs(c.Connectors.GetServices()) {
		svc := c.Connectors.GetServices()[id]
		if issue, ok := failingChatBotIssue(id, svc); ok {
			issues = append(issues, issue)
		}
	}
	return issues, nil
}

func failingChatBotIssue(id string, svc connector.BaseConnector) (armhealth.Issue, bool) {
	bot := connectorChatBot(svc)
	if bot == nil || strings.TrimSpace(bot.ErrorMessage) == "" {
		return armhealth.Issue{}, false
	}
	return armhealth.Issue{
		Key:      fmt.Sprintf("bot:%s", id),
		Severity: armhealth.SeverityWarning,
		Title:    fmt.Sprintf("Chat bot %q is failing", chatBotLabel(bot)),
		Detail:   bot.ErrorMessage,
		Link:     chatBotLink(bot),
	}, true
}

func connectorChatBot(svc connector.BaseConnector) *connector.ChatBot {
	botSvc, ok := svc.(connector.ConnectorWithChatBot)
	if !ok || botSvc == nil {
		return nil
	}
	return botSvc.GetChatBot()
}

func chatBotLabel(bot *connector.ChatBot) string {
	if name := strings.TrimSpace(bot.Name); name != "" {
		return name
	}
	if bot.BotID != "" {
		return bot.BotID
	}
	return bot.Connector
}

func chatBotLink(bot *connector.ChatBot) string {
	if bot.Connector != "" && bot.BotID != "" {
		return fmt.Sprintf("/chat-bots/%s/%s", bot.Connector, bot.BotID)
	}
	return "/chat-bots"
}

func sortedConnectorIDs(services map[string]connector.BaseConnector) []string {
	ids := make([]string, 0, len(services))
	for id := range services {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}
