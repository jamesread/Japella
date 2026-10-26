package db

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
)

// FeedColumn is an extra column a user shows beside the main feed: recent
// posts matching a hashtag or search query, fetched through one or more of
// their social accounts.
type FeedColumn struct {
	ID            uint32       `db:"id"`
	UserAccountID uint32       `db:"user_account_id"`
	Title         string       `db:"title"`
	Query         string       `db:"query"`
	SortOrder     uint32       `db:"sort_order"`
	CreatedAt     sql.NullTime `db:"created_at"`
	UpdatedAt     sql.NullTime `db:"updated_at"`

	// SocialAccountIDs is stored in feed_column_social_accounts.
	SocialAccountIDs []uint32 `db:"-"`
}

// loadFeedColumnAccounts fills SocialAccountIDs for the given columns.
func (db *DB) loadFeedColumnAccounts(cols []*FeedColumn) error {
	if len(cols) == 0 {
		return nil
	}
	byID := make(map[uint32]*FeedColumn, len(cols))
	placeholders := make([]string, 0, len(cols))
	args := make([]any, 0, len(cols))
	for _, c := range cols {
		byID[c.ID] = c
		placeholders = append(placeholders, "?")
		args = append(args, c.ID)
	}

	rows := make([]struct {
		FeedColumnID    uint32 `db:"feed_column_id"`
		SocialAccountID uint32 `db:"social_account_id"`
	}, 0)
	err := db.ResilientSelect(&rows, `
		SELECT feed_column_id, social_account_id FROM feed_column_social_accounts
		WHERE feed_column_id IN (`+strings.Join(placeholders, ",")+`)
		ORDER BY feed_column_id, social_account_id`, args...)
	if err != nil {
		db.Logger().Errorf("Failed to select feed column accounts: %v", err)
		return err
	}
	for _, r := range rows {
		if c := byID[r.FeedColumnID]; c != nil {
			c.SocialAccountIDs = append(c.SocialAccountIDs, r.SocialAccountID)
		}
	}
	return nil
}

func setFeedColumnAccounts(tx *sqlx.Tx, columnID uint32, accountIDs []uint32) error {
	if _, err := tx.Exec(`DELETE FROM feed_column_social_accounts WHERE feed_column_id = ?`, columnID); err != nil {
		return err
	}
	for _, id := range accountIDs {
		if _, err := tx.Exec(`INSERT INTO feed_column_social_accounts (feed_column_id, social_account_id) VALUES (?, ?)`, columnID, id); err != nil {
			return err
		}
	}
	return nil
}

// SelectFeedColumnsForUser returns a user's columns, left to right.
func (db *DB) SelectFeedColumnsForUser(userID uint32) ([]*FeedColumn, error) {
	cols := make([]*FeedColumn, 0)
	err := db.ResilientSelect(&cols, `SELECT * FROM feed_columns WHERE user_account_id = ? ORDER BY sort_order, id`, userID)
	if err != nil {
		db.Logger().Errorf("Failed to select feed columns for user %d: %v", userID, err)
		return nil, err
	}
	if err := db.loadFeedColumnAccounts(cols); err != nil {
		return nil, err
	}
	return cols, nil
}

// GetFeedColumn returns one column, or nil if it does not exist.
func (db *DB) GetFeedColumn(id uint32) (*FeedColumn, error) {
	var col FeedColumn
	err := db.ResilientGet(&col, `SELECT * FROM feed_columns WHERE id = ?`, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		db.Logger().Errorf("Failed to get feed column %d: %v", id, err)
		return nil, err
	}
	if err := db.loadFeedColumnAccounts([]*FeedColumn{&col}); err != nil {
		return nil, err
	}
	return &col, nil
}

// CreateFeedColumn appends a column after the user's existing ones.
func (db *DB) CreateFeedColumn(col *FeedColumn) (uint32, error) {
	if db.connx == nil {
		db.ReconnectDatabaseAndSetErrorMessage()
		return 0, fmt.Errorf("database connection is not established")
	}
	tx, err := db.connx.Beginx()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	res, err := tx.Exec(`
		INSERT INTO feed_columns (user_account_id, title, query, sort_order, created_at, updated_at)
		SELECT ?, ?, ?, COALESCE(MAX(sort_order) + 1, 0), NOW(3), NOW(3)
		FROM feed_columns WHERE user_account_id = ?`,
		col.UserAccountID, col.Title, col.Query, col.UserAccountID)
	if err != nil {
		db.Logger().Errorf("Failed to create feed column: %v", err)
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	if err := setFeedColumnAccounts(tx, uint32(id), col.SocialAccountIDs); err != nil {
		db.Logger().Errorf("Failed to set accounts for feed column %d: %v", id, err)
		return 0, err
	}
	return uint32(id), tx.Commit()
}

// UpdateFeedColumn changes a column's title, query and accounts.
func (db *DB) UpdateFeedColumn(col *FeedColumn) error {
	if db.connx == nil {
		db.ReconnectDatabaseAndSetErrorMessage()
		return fmt.Errorf("database connection is not established")
	}
	tx, err := db.connx.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	res, err := tx.Exec(`
		UPDATE feed_columns SET title = ?, query = ?, updated_at = NOW(3)
		WHERE id = ? AND user_account_id = ?`,
		col.Title, col.Query, col.ID, col.UserAccountID)
	if err != nil {
		db.Logger().Errorf("Failed to update feed column %d: %v", col.ID, err)
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		// Either not this user's column, or nothing changed; check before touching accounts.
		var owner uint32
		if err := tx.Get(&owner, `SELECT user_account_id FROM feed_columns WHERE id = ?`, col.ID); err != nil || owner != col.UserAccountID {
			return fmt.Errorf("feed column %d not found", col.ID)
		}
	}
	if err := setFeedColumnAccounts(tx, col.ID, col.SocialAccountIDs); err != nil {
		db.Logger().Errorf("Failed to set accounts for feed column %d: %v", col.ID, err)
		return err
	}
	return tx.Commit()
}

// DeleteFeedColumn removes one of a user's columns.
func (db *DB) DeleteFeedColumn(userID, id uint32) error {
	_, err := db.ResilientExec(`DELETE FROM feed_columns WHERE id = ? AND user_account_id = ?`, id, userID)
	if err != nil {
		db.Logger().Errorf("Failed to delete feed column %d: %v", id, err)
	}
	return err
}

// ReorderFeedColumns sets the user's column order to ids, left to right.
// Columns not listed keep their place after the listed ones.
func (db *DB) ReorderFeedColumns(userID uint32, ids []uint32) error {
	if db.connx == nil {
		db.ReconnectDatabaseAndSetErrorMessage()
		return fmt.Errorf("database connection is not established")
	}
	tx, err := db.connx.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err = tx.Exec(`UPDATE feed_columns SET sort_order = sort_order + ? WHERE user_account_id = ?`, len(ids), userID); err != nil {
		return err
	}
	for i, id := range ids {
		if _, err = tx.Exec(`UPDATE feed_columns SET sort_order = ?, updated_at = NOW(3) WHERE id = ? AND user_account_id = ?`, i, id, userID); err != nil {
			db.Logger().Errorf("Failed to reorder feed column %d: %v", id, err)
			return err
		}
	}
	return tx.Commit()
}
