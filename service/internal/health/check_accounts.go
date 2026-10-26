package health

import (
	"context"
	"fmt"
	"strings"
	"time"

	armhealth "github.com/jamesread/armature-health"
	"github.com/jamesread/japella/internal/db"
)

// KeywordWatchesFailingCheck finds social accounts with enabled keyword watches whose last run errored.
type KeywordWatchesFailingCheck struct {
	DB *db.DB
}

func (c KeywordWatchesFailingCheck) ID() string    { return "keyword_watches.failing" }
func (c KeywordWatchesFailingCheck) Title() string { return "Failing keyword watches" }

type failingKeywordWatchRow struct {
	SocialAccountID uint32 `db:"social_account_id"`
	Identity        string `db:"identity"`
	Connector       string `db:"connector"`
	LastError       string `db:"last_error"`
}

func (c KeywordWatchesFailingCheck) Run(_ context.Context) ([]armhealth.Issue, error) {
	if c.DB == nil {
		return nil, nil
	}
	var rows []failingKeywordWatchRow
	err := c.DB.ResilientSelect(&rows, `
		SELECT kws.social_account_id, COALESCE(sa.identity, '') AS identity,
		       COALESCE(sa.connector, '') AS connector, kws.last_error
		FROM keyword_watch_state kws
		INNER JOIN social_accounts sa ON sa.id = kws.social_account_id
		WHERE kws.last_error <> ''
		  AND EXISTS (
			SELECT 1 FROM keyword_watches kw
			WHERE kw.social_account_id = kws.social_account_id AND kw.enabled = 1
		  )
		ORDER BY kws.social_account_id`)
	if err != nil {
		return nil, err
	}
	return failingKeywordWatchIssues(rows), nil
}

func failingKeywordWatchIssues(rows []failingKeywordWatchRow) []armhealth.Issue {
	var issues []armhealth.Issue
	for _, row := range rows {
		label := strings.TrimSpace(row.Identity)
		if label == "" {
			label = fmt.Sprintf("#%d", row.SocialAccountID)
		}
		issues = append(issues, armhealth.Issue{
			Key:      fmt.Sprintf("watch_account:%d", row.SocialAccountID),
			Severity: armhealth.SeverityWarning,
			Title:    fmt.Sprintf("Keyword watch for %q (%s) is failing", label, row.Connector),
			Detail:   row.LastError,
			Link:     fmt.Sprintf("/social-accounts/%d", row.SocialAccountID),
		})
	}
	return issues
}

// SocialAccountsExpiredTokensCheck finds active social accounts whose OAuth token is expired past the auto-refresh window.
type SocialAccountsExpiredTokensCheck struct {
	DB  *db.DB
	Now func() time.Time
}

func (c SocialAccountsExpiredTokensCheck) ID() string    { return "social_accounts.expired_tokens" }
func (c SocialAccountsExpiredTokensCheck) Title() string { return "Expired social account tokens" }

func (c SocialAccountsExpiredTokensCheck) Run(_ context.Context) ([]armhealth.Issue, error) {
	if c.DB == nil {
		return nil, nil
	}
	var accounts []*db.SocialAccount
	if err := c.DB.ResilientSelect(&accounts, "SELECT * FROM social_accounts WHERE active = 1 ORDER BY id"); err != nil {
		return nil, err
	}
	now := time.Now()
	if c.Now != nil {
		now = c.Now()
	}
	return expiredAccountTokenIssues(accounts, now), nil
}

func expiredAccountTokenIssues(accounts []*db.SocialAccount, now time.Time) []armhealth.Issue {
	var issues []armhealth.Issue
	for _, sa := range accounts {
		if issue, ok := expiredAccountTokenIssue(sa, now); ok {
			issues = append(issues, issue)
		}
	}
	return issues
}

func expiredAccountTokenIssue(sa *db.SocialAccount, now time.Time) (armhealth.Issue, bool) {
	if !isTokenPastAutoRefreshWindow(sa, now) {
		return armhealth.Issue{}, false
	}
	when := sa.OAuth2TokenExpiry.UTC().Format("2006-01-02 15:04 MST")
	return armhealth.Issue{
		Key:      fmt.Sprintf("account_token:%d", sa.ID),
		Severity: armhealth.SeverityWarning,
		Title:    fmt.Sprintf("Social account %q (%s) has an expired OAuth token", sa.Identity, sa.Connector),
		Detail:   fmt.Sprintf("Access token expired at %s; refresh or reconnect the account.", when),
		Link:     fmt.Sprintf("/social-accounts/%d", sa.ID),
	}, true
}

func isTokenPastAutoRefreshWindow(sa *db.SocialAccount, now time.Time) bool {
	if sa == nil || !sa.Active {
		return false
	}
	return sa.OAuth2TokenExpiry.Year() > 1971 && now.Sub(sa.OAuth2TokenExpiry) > 2*time.Hour
}
