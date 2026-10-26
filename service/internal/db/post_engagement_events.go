package db

import (
	"database/sql"
	"time"
)

// PostEngagementEvent records an increase in a post's engagement counters,
// detected by one run of the post_engagement job.
type PostEngagementEvent struct {
	ID              uint32    `db:"id"`
	PostID          uint32    `db:"post_id"`
	SocialAccountID uint32    `db:"social_account_id"`
	Likes           uint32    `db:"likes"`
	Reposts         uint32    `db:"reposts"`
	Replies         uint32    `db:"replies"`
	Quotes          uint32    `db:"quotes"`
	DetectedAt      time.Time `db:"detected_at"`

	// Joined from posts.
	PostContent string         `db:"post_content"`
	PostURL     sql.NullString `db:"post_url"`
}

// InsertPostEngagementEvent stores one detected increase. detected_at is
// written from Go in UTC (like post_replies.posted_at) so both can be
// compared against the same cutoff.
func (db *DB) InsertPostEngagementEvent(ev *PostEngagementEvent) error {
	_, err := db.ResilientExec(`
		INSERT INTO post_engagement_events (post_id, social_account_id, likes, reposts, replies, quotes, detected_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		ev.PostID, ev.SocialAccountID, ev.Likes, ev.Reposts, ev.Replies, ev.Quotes, time.Now().UTC())
	if err != nil {
		db.Logger().Errorf("Failed to insert engagement event for post %d: %v", ev.PostID, err)
	}
	return err
}

// DeletePostEngagementEventsBefore prunes events older than cutoff.
func (db *DB) DeletePostEngagementEventsBefore(cutoff time.Time) error {
	_, err := db.ResilientExec(`DELETE FROM post_engagement_events WHERE detected_at < ?`, cutoff)
	if err != nil {
		db.Logger().Errorf("Failed to prune engagement events: %v", err)
	}
	return err
}

// SelectPostEngagementEventsForAccountSince returns events for an account
// detected at or after since, newest first.
func (db *DB) SelectPostEngagementEventsForAccountSince(socialAccountID uint32, since time.Time) ([]*PostEngagementEvent, error) {
	events := make([]*PostEngagementEvent, 0)
	err := db.ResilientSelect(&events, `
		SELECT e.id, e.post_id, e.social_account_id, e.likes, e.reposts, e.replies, e.quotes, e.detected_at,
			p.content AS post_content, p.post_url AS post_url
		FROM post_engagement_events e
		INNER JOIN posts p ON p.id = e.post_id
		WHERE e.social_account_id = ? AND e.detected_at >= ?
		ORDER BY e.detected_at DESC, e.id DESC`, socialAccountID, since)
	if err != nil {
		db.Logger().Errorf("Failed to select engagement events for account %d: %v", socialAccountID, err)
		return nil, err
	}
	return events, nil
}

// AccountPostReply is a stored reply joined with the post it belongs to.
type AccountPostReply struct {
	PostReply
	PostContent string         `db:"post_content"`
	PostURL     sql.NullString `db:"post_url"`
}

// SelectPostRepliesForAccountSince returns replies to an account's posts
// posted at or after since, newest first.
func (db *DB) SelectPostRepliesForAccountSince(socialAccountID uint32, since time.Time) ([]*AccountPostReply, error) {
	replies := make([]*AccountPostReply, 0)
	err := db.ResilientSelect(&replies, `
		SELECT r.*, p.content AS post_content, p.post_url AS post_url
		FROM post_replies r
		INNER JOIN posts p ON p.id = r.post_id
		WHERE p.social_account_id = ? AND r.posted_at >= ?
		ORDER BY r.posted_at DESC, r.id DESC`, socialAccountID, since)
	if err != nil {
		db.Logger().Errorf("Failed to select replies for account %d: %v", socialAccountID, err)
		return nil, err
	}
	return replies, nil
}

// SelectCompletedPostsForAccountWithin returns completed posts for an account
// published within the last window, newest first.
func (db *DB) SelectCompletedPostsForAccountWithin(socialAccountID uint32, window time.Duration) ([]*Post, error) {
	posts := make([]*Post, 0)
	err := db.ResilientSelect(&posts, `
		SELECT p.id, p.social_account_id, p.status, p.state, p.content, p.post_url, p.remote_id,
			p.scheduled_at, p.created_at, p.campaign_id AS campaign_id
		FROM posts p
		WHERE p.social_account_id = ?
			AND p.state = ?
			AND COALESCE(p.scheduled_at, p.created_at) >= NOW() - INTERVAL ? SECOND
		ORDER BY COALESCE(p.scheduled_at, p.created_at) DESC, p.id DESC`,
		socialAccountID, PostStateCompleted, int64(window/time.Second))
	if err != nil {
		db.Logger().Errorf("Failed to select posts for account %d: %v", socialAccountID, err)
		return nil, err
	}
	return posts, nil
}
