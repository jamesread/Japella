package db

import (
	"database/sql"
	"fmt"
	"time"
)

// PostReply is a stored reply in the conversation under a published post.
type PostReply struct {
	ID              uint32       `db:"id"`
	PostID          uint32       `db:"post_id"`
	RemoteID        string       `db:"remote_id"`
	ParentRemoteID  string       `db:"parent_remote_id"`
	RemoteURL       string       `db:"remote_url"`
	AuthorName      string       `db:"author_name"`
	AuthorHandle    string       `db:"author_handle"`
	AuthorAvatarURL string       `db:"author_avatar_url"`
	Content         string       `db:"content"`
	PostedAt        sql.NullTime `db:"posted_at"`
	CreatedAt       sql.NullTime `db:"created_at"`
}

// ReplacePostReplies swaps the stored replies for a post with a fresh fetch
// and records the replies counter they were fetched at.
func (db *DB) ReplacePostReplies(postID uint32, replies []*PostReply, replyCount uint32) error {
	if db.connx == nil {
		db.ReconnectDatabaseAndSetErrorMessage()
		return fmt.Errorf("database connection is not established")
	}
	tx, err := db.connx.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err = tx.Exec(`DELETE FROM post_replies WHERE post_id = ?`, postID); err != nil {
		db.Logger().Errorf("Failed to clear replies for post %d: %v", postID, err)
		return err
	}

	for _, r := range replies {
		var postedAt any
		if r.PostedAt.Valid {
			postedAt = r.PostedAt.Time
		}
		if _, err = tx.Exec(`
			INSERT INTO post_replies
				(post_id, remote_id, parent_remote_id, remote_url, author_name, author_handle, author_avatar_url, content, posted_at, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, NOW(3))`,
			postID, r.RemoteID, r.ParentRemoteID, r.RemoteURL, r.AuthorName, r.AuthorHandle, r.AuthorAvatarURL, r.Content, postedAt); err != nil {
			db.Logger().Errorf("Failed to store reply for post %d: %v", postID, err)
			return err
		}
	}

	if _, err = tx.Exec(`
		INSERT INTO post_engagement (post_id, replies_fetched_at, replies_fetched_count, created_at, updated_at)
		VALUES (?, NOW(3), ?, NOW(3), NOW(3))
		ON DUPLICATE KEY UPDATE replies_fetched_at = NOW(3), replies_fetched_count = VALUES(replies_fetched_count), updated_at = NOW(3)`,
		postID, replyCount); err != nil {
		db.Logger().Errorf("Failed to record replies fetch for post %d: %v", postID, err)
		return err
	}

	return tx.Commit()
}

// SelectPostReplies returns the stored replies for a post, oldest first.
func (db *DB) SelectPostReplies(postID uint32) ([]*PostReply, error) {
	replies := make([]*PostReply, 0)
	err := db.ResilientSelect(&replies, `
		SELECT * FROM post_replies
		WHERE post_id = ?
		ORDER BY posted_at IS NULL, posted_at ASC, id ASC`, postID)
	if err != nil {
		db.Logger().Errorf("Failed to select replies for post %d: %v", postID, err)
		return nil, err
	}
	return replies, nil
}

// NullTimeOf wraps t, treating the zero time as NULL.
func NullTimeOf(t time.Time) sql.NullTime {
	return sql.NullTime{Time: t, Valid: !t.IsZero()}
}
