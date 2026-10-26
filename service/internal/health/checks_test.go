package health

import (
	"context"
	"strings"
	"testing"
	"time"

	armhealth "github.com/jamesread/armature-health"
	"github.com/jamesread/armature-health/errorlog"
	"github.com/jamesread/japella/internal/connector"
	"github.com/jamesread/japella/internal/db"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

type stubConnectorSource struct {
	services map[string]connector.BaseConnector
}

func (s stubConnectorSource) GetServices() map[string]connector.BaseConnector {
	return s.services
}

type stubConnector struct {
	protocol string
	issues   []connector.ConfigurationIssue
	bot      *connector.ChatBot
}

func (c *stubConnector) SetStartupConfiguration(*connector.ControllerStartupConfiguration) {}
func (c *stubConnector) Start()                                                            {}
func (c *stubConnector) GetIdentity() string                                               { return c.protocol }
func (c *stubConnector) GetProtocol() string                                               { return c.protocol }
func (c *stubConnector) GetIcon() string                                                   { return c.protocol }
func (c *stubConnector) OnRefresh(*db.SocialAccount) error                                 { return nil }
func (c *stubConnector) GetCvars() map[string]*db.Cvar                                     { return nil }
func (c *stubConnector) OnCvarChange(string, string)                                       {}
func (c *stubConnector) CheckConfiguration() *connector.ConfigurationCheckResult {
	return &connector.ConfigurationCheckResult{Issues: c.issues}
}
func (c *stubConnector) GetChatBot() *connector.ChatBot {
	return c.bot
}

func newTestSQLiteDB(t *testing.T) *db.DB {
	t.Helper()
	connx, err := sqlx.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = connx.Close() })

	schema := `
		CREATE TABLE social_accounts (
			id INTEGER PRIMARY KEY,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			connector TEXT NOT NULL,
			identity TEXT NOT NULL,
			did TEXT NOT NULL DEFAULT '',
			homeserver TEXT NOT NULL DEFAULT '',
			oauth2_token TEXT NOT NULL DEFAULT '',
			oauth2_token_expiry TIMESTAMP NOT NULL DEFAULT '1970-01-01 00:00:00',
			oauth2_refresh_token TEXT NOT NULL DEFAULT '',
			dpop_key TEXT,
			active INTEGER NOT NULL DEFAULT 1,
			state TEXT NOT NULL DEFAULT 'active',
			owner_user_id INTEGER
		);
		CREATE TABLE keyword_watches (
			id INTEGER PRIMARY KEY,
			social_account_id INTEGER NOT NULL,
			keyword TEXT NOT NULL,
			keyword_normalized TEXT NOT NULL,
			enabled INTEGER NOT NULL DEFAULT 1
		);
		CREATE TABLE keyword_watch_state (
			social_account_id INTEGER PRIMARY KEY,
			last_error TEXT NOT NULL DEFAULT ''
		);
		CREATE TABLE health_issues (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			check_id TEXT NOT NULL,
			issue_key TEXT NOT NULL,
			severity TEXT NOT NULL,
			title TEXT NOT NULL,
			detail TEXT NOT NULL,
			link TEXT NOT NULL,
			first_seen INTEGER NOT NULL,
			last_seen INTEGER NOT NULL,
			occurrences INTEGER NOT NULL DEFAULT 1,
			resolved_at INTEGER NULL,
			acknowledged_at INTEGER NULL,
			acknowledged_by INTEGER NULL
		);
		CREATE TABLE health_check_runs (
			check_id TEXT NOT NULL PRIMARY KEY,
			last_run_at INTEGER NOT NULL,
			duration_ms INTEGER NOT NULL,
			status TEXT NOT NULL,
			error TEXT NOT NULL
		);
	`
	if _, err := connx.Exec(schema); err != nil {
		t.Fatalf("create schema: %v", err)
	}
	database := &db.DB{}
	database.SetSQLxDB(connx)
	return database
}

func runCheck(t *testing.T, check armhealth.Check) map[string]armhealth.Issue {
	t.Helper()
	issues, err := check.Run(context.Background())
	if err != nil {
		t.Fatalf("%s: %v", check.ID(), err)
	}
	byKey := map[string]armhealth.Issue{}
	for _, issue := range issues {
		byKey[issue.Key] = issue
	}
	return byKey
}

func TestDefaultChecksHaveUniqueIDs(t *testing.T) {
	checks := DefaultChecks(Deps{
		DB:                       newTestSQLiteDB(t),
		ErrorLog:                 errorlog.New(),
		AdminUsesDefaultPassword: func(context.Context) (bool, error) { return false, nil },
		Getenv:                   func(string) string { return "" },
	})
	seen := map[string]bool{}
	for _, check := range checks {
		if seen[check.ID()] || check.Title() == "" {
			t.Fatalf("check %q is duplicated or untitled", check.ID())
		}
		seen[check.ID()] = true
	}
	if len(checks) != 7 {
		t.Fatalf("expected 7 default checks, got %d", len(checks))
	}
}

func TestRiskySettingsWiring(t *testing.T) {
	env := map[string]string{"JAPELLA_RESET_ADMIN_PASSWORD": "1"}
	var risky armhealth.Check
	for _, check := range DefaultChecks(Deps{Getenv: func(k string) string { return env[k] }, LogLevel: "debug"}) {
		if check.ID() == "config.risky_settings" {
			risky = check
		}
	}
	if risky == nil {
		t.Fatal("config.risky_settings not registered")
	}
	got := runCheck(t, risky)
	if _, ok := got["env:JAPELLA_RESET_ADMIN_PASSWORD"]; !ok {
		t.Fatalf("missing env:JAPELLA_RESET_ADMIN_PASSWORD in %v", got)
	}
	if _, ok := got["log-level"]; !ok {
		t.Fatalf("missing log-level in %v", got)
	}
}

func TestConnectorAndChatBotChecks(t *testing.T) {
	src := stubConnectorSource{
		services: map[string]connector.BaseConnector{
			"mastodon": &stubConnector{
				protocol: "mastodon",
				issues: []connector.ConfigurationIssue{
					{Message: "Missing client secret", FixPath: "/settings", FixHash: "mastodon_secret"},
				},
			},
			"telegram:bot1": &stubConnector{
				protocol: "telegram",
				bot: &connector.ChatBot{
					BotID:        "bot1",
					Connector:    "telegram",
					Name:         "Support Bot",
					ErrorMessage: "unauthorized token",
				},
			},
		},
	}

	cfgGot := runCheck(t, ConnectorConfigCheck{Connectors: src})
	issue, ok := cfgGot["connector:mastodon:0"]
	if !ok || issue.Link != "/settings#mastodon_secret" {
		t.Fatalf("unexpected connector config issue: %+v", cfgGot)
	}

	botGot := runCheck(t, ChatBotsFailingCheck{Connectors: src})
	botIssue, ok := botGot["bot:telegram:bot1"]
	if !ok || botIssue.Link != "/chat-bots/telegram/bot1" || !strings.Contains(botIssue.Title, "Support Bot") {
		t.Fatalf("unexpected chatbot issue: %+v", botGot)
	}
}

func TestAccountAndKeywordWatchChecks(t *testing.T) {
	database := newTestSQLiteDB(t)
	now := time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)
	expired := now.Add(-4 * time.Hour).Format("2006-01-02 15:04:05")

	_, err := database.ResilientExec(`
		INSERT INTO social_accounts (id, connector, identity, oauth2_token_expiry, active)
		VALUES (10, 'mastodon', '@alice@example.social', ?, 1),
		       (11, 'bluesky', 'bob.bsky.social', '1970-01-01 00:00:00', 1);
		INSERT INTO keyword_watches (id, social_account_id, keyword, keyword_normalized, enabled)
		VALUES (1, 10, 'golang', 'golang', 1);
		INSERT INTO keyword_watch_state (social_account_id, last_error)
		VALUES (10, 'HTTP 429 rate limited'),
		       (11, 'ignored because no enabled watches');
	`, expired)
	if err != nil {
		t.Fatalf("seed: %v", err)
	}

	watchGot := runCheck(t, KeywordWatchesFailingCheck{DB: database})
	if len(watchGot) != 1 || watchGot["watch_account:10"].Link != "/social-accounts/10" {
		t.Fatalf("unexpected keyword watch issues: %+v", watchGot)
	}

	tokenGot := runCheck(t, SocialAccountsExpiredTokensCheck{
		DB:  database,
		Now: func() time.Time { return now },
	})
	if len(tokenGot) != 1 || tokenGot["account_token:10"].Link != "/social-accounts/10" {
		t.Fatalf("unexpected expired token issues: %+v", tokenGot)
	}
}

func TestServiceIntegrationWithDBStore(t *testing.T) {
	database := newTestSQLiteDB(t)
	st := NewDBStore(database, "sqlite3")
	svc := armhealth.NewService(st, DefaultChecks(Deps{
		DB:                       database,
		ErrorLog:                 errorlog.New(),
		AdminUsesDefaultPassword: func(context.Context) (bool, error) { return true, nil },
		Getenv:                   func(string) string { return "" },
	}), WebhookNotifier{}, armhealth.Options{})

	svc.RunAll(context.Background())
	statuses, err := svc.Statuses(context.Background())
	if err != nil {
		t.Fatalf("statuses: %v", err)
	}
	if len(statuses) != 7 {
		t.Fatalf("expected 7 check statuses, got %d", len(statuses))
	}
	summary, err := svc.Summary(context.Background())
	if err != nil {
		t.Fatalf("summary: %v", err)
	}
	if summary.Errors != 1 {
		t.Fatalf("expected 1 error (default admin password), got %+v", summary)
	}
}
