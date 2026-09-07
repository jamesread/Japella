package db

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

const (
	WatchModeEvaluating   = "evaluating"
	WatchModeNative       = "native"
	WatchModeSearch       = "search"
	WatchModeDegraded     = "degraded"

	WatchFidelityExactRule    = "exact_rule"
	WatchFidelityClientFilter = "client_filter"
	WatchFidelityHashtagOnly  = "hashtag_only"
	WatchFidelitySearchOnly   = "search_only"
)

// KeywordWatch is a keyword subscription bound to a social account.
type KeywordWatch struct {
	Model
	SocialAccountID    uint32 `db:"social_account_id"`
	Keyword            string `db:"keyword"`
	KeywordNormalized  string `db:"keyword_normalized"`
	Enabled            bool   `db:"enabled"`
}

// KeywordWatchState is runtime acquisition state for an account's watches.
type KeywordWatchState struct {
	SocialAccountID  uint32         `db:"social_account_id"`
	AcquisitionMode  string         `db:"acquisition_mode"`
	NativeFidelity   string         `db:"native_fidelity"`
	BackoffUntil     sql.NullTime   `db:"backoff_until"`
	LastSuccessAt    sql.NullTime   `db:"last_success_at"`
	LastErrorAt      sql.NullTime   `db:"last_error_at"`
	LastError        string         `db:"last_error"`
	LastPollAt       sql.NullTime   `db:"last_poll_at"`
	EmitsThisHour    int            `db:"emits_this_hour"`
	EmitsHourBucket  sql.NullTime   `db:"emits_hour_bucket"`
	UpdatedAt        time.Time      `db:"updated_at"`
}

// KeywordMatch is a deduped observation of a remote post matching a watch.
type KeywordMatch struct {
	ID                   uint32       `db:"id"`
	SocialAccountID      uint32       `db:"social_account_id"`
	KeywordWatchID       uint32       `db:"keyword_watch_id"`
	Connector            string       `db:"connector"`
	RemoteID             string       `db:"remote_id"`
	RemoteURL            string       `db:"remote_url"`
	AuthorID             string       `db:"author_id"`
	AuthorHandle         string       `db:"author_handle"`
	ContentSnippet       string       `db:"content_snippet"`
	MatchedAt            time.Time    `db:"matched_at"`
	PostedAt             sql.NullTime `db:"posted_at"`
	AcquisitionMode      string       `db:"acquisition_mode"`
	WebhookDispatchedAt  sql.NullTime `db:"webhook_dispatched_at"`
	CreatedAt            time.Time    `db:"created_at"`
}

func (db *DB) ListKeywordWatches(socialAccountID uint32) ([]*KeywordWatch, error) {
	out := make([]*KeywordWatch, 0)
	err := db.ResilientSelect(&out, `
		SELECT id, social_account_id, keyword, keyword_normalized, enabled, created_at, updated_at
		FROM keyword_watches
		WHERE social_account_id = ?
		ORDER BY keyword_normalized`, socialAccountID)
	if err != nil {
		db.Logger().Errorf("ListKeywordWatches: %v", err)
		return nil, err
	}
	return out, nil
}

func (db *DB) ListEnabledKeywordWatches() ([]*KeywordWatch, error) {
	out := make([]*KeywordWatch, 0)
	err := db.ResilientSelect(&out, `
		SELECT id, social_account_id, keyword, keyword_normalized, enabled, created_at, updated_at
		FROM keyword_watches
		WHERE enabled = 1
		ORDER BY social_account_id, keyword_normalized`)
	if err != nil {
		db.Logger().Errorf("ListEnabledKeywordWatches: %v", err)
		return nil, err
	}
	return out, nil
}

func (db *DB) CountKeywordWatches(socialAccountID uint32) (int, error) {
	var n int
	err := db.ResilientGet(&n, `SELECT COUNT(*) FROM keyword_watches WHERE social_account_id = ?`, socialAccountID)
	if err != nil {
		return 0, err
	}
	return n, nil
}

func (db *DB) GetKeywordWatch(id uint32) (*KeywordWatch, error) {
	var w KeywordWatch
	err := db.ResilientGet(&w, `
		SELECT id, social_account_id, keyword, keyword_normalized, enabled, created_at, updated_at
		FROM keyword_watches WHERE id = ?`, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		db.Logger().Errorf("GetKeywordWatch: %v", err)
		return nil, err
	}
	return &w, nil
}

func (db *DB) CreateKeywordWatch(socialAccountID uint32, keyword, keywordNormalized string, enabled bool) (uint32, error) {
	res, err := db.ResilientExec(`
		INSERT INTO keyword_watches (social_account_id, keyword, keyword_normalized, enabled, created_at, updated_at)
		VALUES (?, ?, ?, ?, NOW(3), NOW(3))`,
		socialAccountID, keyword, keywordNormalized, enabled)
	if err != nil {
		db.Logger().Errorf("CreateKeywordWatch: %v", err)
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	if err := db.EnsureKeywordWatchState(socialAccountID); err != nil {
		return uint32(id), err
	}
	return uint32(id), nil
}

func (db *DB) UpdateKeywordWatch(id uint32, keyword, keywordNormalized string, enabled bool) error {
	_, err := db.ResilientExec(`
		UPDATE keyword_watches
		SET
			created_at = IF(keyword_normalized <> ?, NOW(3), created_at),
			keyword = ?,
			keyword_normalized = ?,
			enabled = ?,
			updated_at = NOW(3)
		WHERE id = ?`,
		keywordNormalized, keyword, keywordNormalized, enabled, id)
	if err != nil {
		db.Logger().Errorf("UpdateKeywordWatch: %v", err)
	}
	return err
}

func (db *DB) DeleteKeywordWatch(id uint32) error {
	_, err := db.ResilientExec(`DELETE FROM keyword_watches WHERE id = ?`, id)
	if err != nil {
		db.Logger().Errorf("DeleteKeywordWatch: %v", err)
	}
	return err
}

func (db *DB) EnsureKeywordWatchState(socialAccountID uint32) error {
	_, err := db.ResilientExec(`
		INSERT INTO keyword_watch_state (social_account_id, acquisition_mode, native_fidelity, updated_at)
		VALUES (?, ?, ?, NOW(3))
		ON DUPLICATE KEY UPDATE social_account_id = social_account_id`,
		socialAccountID, WatchModeEvaluating, WatchFidelitySearchOnly)
	if err != nil {
		db.Logger().Errorf("EnsureKeywordWatchState: %v", err)
	}
	return err
}

func (db *DB) GetKeywordWatchState(socialAccountID uint32) (*KeywordWatchState, error) {
	var st KeywordWatchState
	err := db.ResilientGet(&st, `
		SELECT social_account_id, acquisition_mode, native_fidelity, backoff_until,
		       last_success_at, last_error_at, last_error, last_poll_at,
		       emits_this_hour, emits_hour_bucket, updated_at
		FROM keyword_watch_state WHERE social_account_id = ?`, socialAccountID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		db.Logger().Errorf("GetKeywordWatchState: %v", err)
		return nil, err
	}
	return &st, nil
}

func (db *DB) ListKeywordWatchStates() ([]*KeywordWatchState, error) {
	out := make([]*KeywordWatchState, 0)
	err := db.ResilientSelect(&out, `
		SELECT social_account_id, acquisition_mode, native_fidelity, backoff_until,
		       last_success_at, last_error_at, last_error, last_poll_at,
		       emits_this_hour, emits_hour_bucket, updated_at
		FROM keyword_watch_state`)
	if err != nil {
		db.Logger().Errorf("ListKeywordWatchStates: %v", err)
		return nil, err
	}
	return out, nil
}

func (db *DB) UpsertKeywordWatchState(st *KeywordWatchState) error {
	_, err := db.ResilientExec(`
		INSERT INTO keyword_watch_state (
			social_account_id, acquisition_mode, native_fidelity, backoff_until,
			last_success_at, last_error_at, last_error, last_poll_at,
			emits_this_hour, emits_hour_bucket, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NOW(3))
		ON DUPLICATE KEY UPDATE
			acquisition_mode = VALUES(acquisition_mode),
			native_fidelity = VALUES(native_fidelity),
			backoff_until = VALUES(backoff_until),
			last_success_at = VALUES(last_success_at),
			last_error_at = VALUES(last_error_at),
			last_error = VALUES(last_error),
			last_poll_at = VALUES(last_poll_at),
			emits_this_hour = VALUES(emits_this_hour),
			emits_hour_bucket = VALUES(emits_hour_bucket),
			updated_at = NOW(3)`,
		st.SocialAccountID, st.AcquisitionMode, st.NativeFidelity, st.BackoffUntil,
		st.LastSuccessAt, st.LastErrorAt, st.LastError, st.LastPollAt,
		st.EmitsThisHour, st.EmitsHourBucket)
	if err != nil {
		db.Logger().Errorf("UpsertKeywordWatchState: %v", err)
	}
	return err
}

// TryInsertKeywordMatch inserts a match. Returns (inserted, id, error).
// Duplicate (watch_id, remote_id) returns inserted=false without error.
func (db *DB) TryInsertKeywordMatch(m *KeywordMatch) (bool, uint32, error) {
	res, err := db.ResilientExec(`
		INSERT INTO keyword_matches (
			social_account_id, keyword_watch_id, connector, remote_id, remote_url,
			author_id, author_handle, content_snippet, matched_at, posted_at,
			acquisition_mode, webhook_dispatched_at, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NOW(3))`,
		m.SocialAccountID, m.KeywordWatchID, m.Connector, m.RemoteID, m.RemoteURL,
		m.AuthorID, m.AuthorHandle, m.ContentSnippet, m.MatchedAt, m.PostedAt,
		m.AcquisitionMode, m.WebhookDispatchedAt)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "duplicate") {
			return false, 0, nil
		}
		db.Logger().Errorf("TryInsertKeywordMatch: %v", err)
		return false, 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return false, 0, err
	}
	return true, uint32(id), nil
}

func (db *DB) MarkKeywordMatchWebhookDispatched(id uint32) error {
	_, err := db.ResilientExec(`UPDATE keyword_matches SET webhook_dispatched_at = NOW(3) WHERE id = ?`, id)
	if err != nil {
		db.Logger().Errorf("MarkKeywordMatchWebhookDispatched: %v", err)
	}
	return err
}

func (db *DB) ListRecentKeywordMatches(socialAccountID uint32, limit int) ([]*KeywordMatch, error) {
	if limit <= 0 {
		limit = 20
	}
	out := make([]*KeywordMatch, 0)
	err := db.ResilientSelect(&out, `
		SELECT id, social_account_id, keyword_watch_id, connector, remote_id, remote_url,
		       author_id, author_handle, content_snippet, matched_at, posted_at,
		       acquisition_mode, webhook_dispatched_at, created_at
		FROM keyword_matches
		WHERE social_account_id = ?
		ORDER BY matched_at DESC
		LIMIT ?`, socialAccountID, limit)
	if err != nil {
		db.Logger().Errorf("ListRecentKeywordMatches: %v", err)
		return nil, err
	}
	return out, nil
}

func (db *DB) CleanupOldKeywordMatches(keepPerWatch int) error {
	if keepPerWatch <= 0 {
		keepPerWatch = 200
	}
	all := make([]*KeywordWatch, 0)
	if err := db.ResilientSelect(&all, `SELECT id, social_account_id, keyword, keyword_normalized, enabled, created_at, updated_at FROM keyword_watches`); err != nil {
		db.Logger().Errorf("CleanupOldKeywordMatches list: %v", err)
		return err
	}
	for _, w := range all {
		_, err := db.ResilientExec(`
			DELETE FROM keyword_matches
			WHERE keyword_watch_id = ?
			  AND id NOT IN (
				SELECT id FROM (
					SELECT id FROM keyword_matches
					WHERE keyword_watch_id = ?
					ORDER BY matched_at DESC
					LIMIT ?
				) keep_rows
			)`, w.ID, w.ID, keepPerWatch)
		if err != nil {
			db.Logger().Errorf("CleanupOldKeywordMatches watch %d: %v", w.ID, err)
			return fmt.Errorf("cleanup keyword matches: %w", err)
		}
	}
	return nil
}

func NullTime(t time.Time) sql.NullTime {
	if t.IsZero() {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: t, Valid: true}
}
