package db

import (
	"database/sql"
	"time"
)

// Account activity kinds: things a social account did through Japella.
const (
	AccountActivityLiked     = "liked"
	AccountActivityCommented = "commented"
)

// AccountActivity is one like or comment a social account made through
// Japella, with enough about the post to list it later.
type AccountActivity struct {
	ID              uint32         `db:"id"`
	SocialAccountID uint32         `db:"social_account_id"`
	Kind            string         `db:"kind"`
	PostKeyHash     string         `db:"post_key_hash"`
	LikeKey         sql.NullString `db:"like_key"`
	PostURL         string         `db:"post_url"`
	PostAuthor      string         `db:"post_author"`
	PostSnippet     string         `db:"post_snippet"`
	CommentContent  sql.NullString `db:"comment_content"`
	CommentURL      string         `db:"comment_url"`
	CreatedAt       time.Time      `db:"created_at"`
}

// RecordAccountLike notes that the account liked a post. Liking a post again
// keeps the original time.
func (db *DB) RecordAccountLike(a *AccountActivity) error {
	_, err := db.ResilientExec(`
		INSERT IGNORE INTO account_activity
			(social_account_id, kind, post_key_hash, like_key, post_url, post_author, post_snippet, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, NOW(3))`,
		a.SocialAccountID, AccountActivityLiked, a.PostKeyHash, a.PostKeyHash, a.PostURL, a.PostAuthor, a.PostSnippet)
	if err != nil {
		db.Logger().Errorf("Failed to record like by social account %d: %v", a.SocialAccountID, err)
	}
	return err
}

// DeleteAccountLike forgets a like after the account unlikes the post.
func (db *DB) DeleteAccountLike(socialAccountID uint32, postKeyHash string) error {
	_, err := db.ResilientExec(`
		DELETE FROM account_activity
		WHERE social_account_id = ? AND kind = ? AND like_key = ?`,
		socialAccountID, AccountActivityLiked, postKeyHash)
	if err != nil {
		db.Logger().Errorf("Failed to remove like by social account %d: %v", socialAccountID, err)
	}
	return err
}

// RecordAccountComment notes a comment the account published on a post.
func (db *DB) RecordAccountComment(a *AccountActivity) error {
	_, err := db.ResilientExec(`
		INSERT INTO account_activity
			(social_account_id, kind, post_key_hash, post_url, post_author, post_snippet, comment_content, comment_url, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, NOW(3))`,
		a.SocialAccountID, AccountActivityCommented, a.PostKeyHash, a.PostURL, a.PostAuthor, a.PostSnippet, a.CommentContent, a.CommentURL)
	if err != nil {
		db.Logger().Errorf("Failed to record comment by social account %d: %v", a.SocialAccountID, err)
	}
	return err
}

// SelectAccountActivitySince returns the account's likes and comments made
// since a time, newest first.
func (db *DB) SelectAccountActivitySince(socialAccountID uint32, since time.Time) ([]*AccountActivity, error) {
	rows := make([]*AccountActivity, 0)
	err := db.ResilientSelect(&rows, `
		SELECT * FROM account_activity
		WHERE social_account_id = ? AND created_at >= ?
		ORDER BY created_at DESC, id DESC`, socialAccountID, since)
	if err != nil {
		db.Logger().Errorf("Failed to select activity for account %d: %v", socialAccountID, err)
		return nil, err
	}
	return rows, nil
}
