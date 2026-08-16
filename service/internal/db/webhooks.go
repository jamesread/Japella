package db

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jamesread/japella/internal/webhook"
	"github.com/jmoiron/sqlx"
)

type WebhookTarget struct {
	Model
	URL     string `db:"url"`
	Secret  string `db:"secret"`
	Enabled bool   `db:"enabled"`
	Events  []string
}

func (db *DB) ListWebhookTargets() ([]*WebhookTarget, error) {
	ret := make([]*WebhookTarget, 0)
	err := db.ResilientSelect(&ret, `
		SELECT id, url, secret, enabled, created_at, updated_at
		FROM webhook_targets
		ORDER BY id ASC`)
	if err != nil {
		db.Logger().Errorf("ListWebhookTargets: %v", err)
		return nil, err
	}
	for _, t := range ret {
		events, err := db.selectWebhookEvents(t.ID)
		if err != nil {
			return nil, err
		}
		t.Events = events
	}
	return ret, nil
}

func (db *DB) GetWebhookTarget(id uint32) (*WebhookTarget, error) {
	var t WebhookTarget
	err := db.ResilientGet(&t, `
		SELECT id, url, secret, enabled, created_at, updated_at
		FROM webhook_targets WHERE id = ?`, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		db.Logger().Errorf("GetWebhookTarget: %v", err)
		return nil, err
	}
	events, err := db.selectWebhookEvents(t.ID)
	if err != nil {
		return nil, err
	}
	t.Events = events
	return &t, nil
}

func (db *DB) EnabledTargetsForEvent(ctx context.Context, event string) ([]webhook.Target, error) {
	_ = ctx
	rows := make([]struct {
		ID     uint32 `db:"id"`
		URL    string `db:"url"`
		Secret string `db:"secret"`
	}, 0)
	err := db.ResilientSelect(&rows, `
		SELECT t.id, t.url, t.secret
		FROM webhook_targets t
		INNER JOIN webhook_events e ON e.webhook_target_id = t.id
		WHERE e.event = ? AND t.enabled = 1
		ORDER BY t.id ASC`, event)
	if err != nil {
		db.Logger().Errorf("EnabledTargetsForEvent: %v", err)
		return nil, err
	}
	out := make([]webhook.Target, 0, len(rows))
	for _, row := range rows {
		out = append(out, webhook.Target{
			ID:     row.ID,
			URL:    row.URL,
			Secret: row.Secret,
		})
	}
	return out, nil
}

func (db *DB) CreateWebhookTarget(url, secret string, events []string, enabled bool) (uint32, error) {
	if db.connx == nil {
		db.ReconnectDatabaseAndSetErrorMessage()
		return 0, fmt.Errorf("database connection is not established")
	}
	tx, err := db.connx.Beginx()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	enabledInt := 0
	if enabled {
		enabledInt = 1
	}
	res, err := tx.Exec(`
		INSERT INTO webhook_targets (url, secret, enabled, created_at, updated_at)
		VALUES (?, ?, ?, NOW(3), NOW(3))`, url, secret, enabledInt)
	if err != nil {
		return 0, err
	}
	lastID, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	targetID := uint32(lastID)
	if err := insertWebhookEventsTx(tx, targetID, events); err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return targetID, nil
}

func (db *DB) UpdateWebhookTarget(id uint32, url, secret string, events []string, enabled bool, keepSecret bool) error {
	if db.connx == nil {
		db.ReconnectDatabaseAndSetErrorMessage()
		return fmt.Errorf("database connection is not established")
	}
	tx, err := db.connx.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	enabledInt := 0
	if enabled {
		enabledInt = 1
	}

	if keepSecret {
		if _, err = tx.Exec(`
			UPDATE webhook_targets SET url = ?, enabled = ?, updated_at = NOW(3) WHERE id = ?`,
			url, enabledInt, id); err != nil {
			return err
		}
	} else {
		if _, err = tx.Exec(`
			UPDATE webhook_targets SET url = ?, secret = ?, enabled = ?, updated_at = NOW(3) WHERE id = ?`,
			url, secret, enabledInt, id); err != nil {
			return err
		}
	}

	if _, err = tx.Exec(`DELETE FROM webhook_events WHERE webhook_target_id = ?`, id); err != nil {
		return err
	}
	if err := insertWebhookEventsTx(tx, id, events); err != nil {
		return err
	}
	return tx.Commit()
}

func (db *DB) DeleteWebhookTarget(id uint32) error {
	_, err := db.ResilientExec(`DELETE FROM webhook_targets WHERE id = ?`, id)
	if err != nil {
		db.Logger().Errorf("DeleteWebhookTarget: %v", err)
	}
	return err
}

func (db *DB) selectWebhookEvents(targetID uint32) ([]string, error) {
	events := make([]string, 0)
	err := db.ResilientSelect(&events, `
		SELECT event FROM webhook_events WHERE webhook_target_id = ? ORDER BY event`, targetID)
	if err != nil {
		db.Logger().Errorf("selectWebhookEvents: %v", err)
		return nil, err
	}
	return events, nil
}

func insertWebhookEventsTx(tx *sqlx.Tx, targetID uint32, events []string) error {
	for _, event := range events {
		if _, err := tx.Exec(
			`INSERT INTO webhook_events (webhook_target_id, event) VALUES (?, ?)`,
			targetID, event); err != nil {
			return err
		}
	}
	return nil
}
