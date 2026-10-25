package db

import (
	"database/sql"
	"time"
)

// PostEngagement holds the latest engagement counters fetched for a published post.
type PostEngagement struct {
	PostID        uint32       `db:"post_id"`
	Likes         uint32       `db:"likes"`
	Reposts       uint32       `db:"reposts"`
	Replies       uint32       `db:"replies"`
	Quotes        uint32       `db:"quotes"`
	Impressions   uint32       `db:"impressions"`
	LastFetchedAt sql.NullTime `db:"last_fetched_at"`
	LastChangedAt sql.NullTime `db:"last_changed_at"`
	LastError     string       `db:"last_error"`
	CreatedAt     sql.NullTime `db:"created_at"`
	UpdatedAt     sql.NullTime `db:"updated_at"`
}

// SelectPostsForEngagementFetch returns completed posts with a URL, published
// within maxAge, on active social accounts, least-recently-fetched first.
func (db *DB) SelectPostsForEngagementFetch(maxAge time.Duration, limit int) ([]*Post, error) {
	posts := make([]*Post, 0)
	since := time.Now().Add(-maxAge)
	err := db.ResilientSelect(&posts, `
		SELECT p.id, p.social_account_id, p.status, p.state, p.content, p.post_url, p.remote_id,
			p.scheduled_at, p.created_at, p.campaign_id AS campaign_id
		FROM posts p
		INNER JOIN social_accounts sa ON sa.id = p.social_account_id
		LEFT JOIN post_engagement pe ON pe.post_id = p.id
		WHERE p.state = ?
			AND p.post_url IS NOT NULL AND p.post_url <> ''
			AND sa.active = 1
			AND COALESCE(p.scheduled_at, p.created_at) >= ?
		ORDER BY pe.last_fetched_at IS NOT NULL, pe.last_fetched_at ASC, p.id DESC
		LIMIT ?`, PostStateCompleted, since, limit)
	if err != nil {
		db.Logger().Errorf("Failed to select posts for engagement fetch: %v", err)
		return nil, err
	}
	return posts, nil
}

// GetPostEngagement returns the stored engagement for a post, or nil if never fetched.
func (db *DB) GetPostEngagement(postID uint32) (*PostEngagement, error) {
	var pe PostEngagement
	err := db.ResilientGet(&pe, "SELECT * FROM post_engagement WHERE post_id = ?", postID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		db.Logger().Errorf("Failed to get post engagement for post %d: %v", postID, err)
		return nil, err
	}
	return &pe, nil
}

// UpsertPostEngagement stores fresh counters for a post and clears any previous error.
// changed controls whether last_changed_at is bumped.
func (db *DB) UpsertPostEngagement(pe *PostEngagement, changed bool) error {
	_, err := db.ResilientExec(`
		INSERT INTO post_engagement
			(post_id, likes, reposts, replies, quotes, impressions, last_fetched_at, last_changed_at, last_error, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, NOW(3), NOW(3), '', NOW(3), NOW(3))
		ON DUPLICATE KEY UPDATE
			likes = VALUES(likes),
			reposts = VALUES(reposts),
			replies = VALUES(replies),
			quotes = VALUES(quotes),
			impressions = VALUES(impressions),
			last_fetched_at = NOW(3),
			last_changed_at = IF(?, NOW(3), last_changed_at),
			last_error = '',
			updated_at = NOW(3)`,
		pe.PostID, pe.Likes, pe.Reposts, pe.Replies, pe.Quotes, pe.Impressions, changed)
	if err != nil {
		db.Logger().Errorf("Failed to upsert post engagement for post %d: %v", pe.PostID, err)
	}
	return err
}

// MarkPostEngagementError records a fetch failure without touching stored counters.
func (db *DB) MarkPostEngagementError(postID uint32, message string) error {
	if len(message) > 1024 {
		message = message[:1024]
	}
	_, err := db.ResilientExec(`
		INSERT INTO post_engagement (post_id, last_fetched_at, last_error, created_at, updated_at)
		VALUES (?, NOW(3), ?, NOW(3), NOW(3))
		ON DUPLICATE KEY UPDATE last_fetched_at = NOW(3), last_error = VALUES(last_error), updated_at = NOW(3)`,
		postID, message)
	if err != nil {
		db.Logger().Errorf("Failed to record post engagement error for post %d: %v", postID, err)
	}
	return err
}
