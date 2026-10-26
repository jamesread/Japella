package db

import (
	"fmt"
	"strings"
	"time"
)

// TaggedPost is a post a user has tagged, with the copy of it kept for tag columns.
type TaggedPost struct {
	ID              uint32    `db:"id"`
	PostKeyHash     string    `db:"post_key_hash"`
	PostKey         string    `db:"post_key"`
	SocialAccountID uint32    `db:"social_account_id"`
	Snapshot        string    `db:"snapshot"`
	TaggedAt        time.Time `db:"tagged_at"`
}

// TagCount is one of a user's tags and how many posts carry it.
type TagCount struct {
	Tag   string `db:"tag"`
	Posts uint32 `db:"posts"`
}

// SetPostTags replaces a user's tags on a post, identified by its platform
// key (see feedauthor.PostKey). The snapshot is refreshed; with no tags the
// post is forgotten.
func (db *DB) SetPostTags(userID uint32, keyHash, key string, socialAccountID uint32, snapshot string, tags []string) error {
	if db.connx == nil {
		db.ReconnectDatabaseAndSetErrorMessage()
		return fmt.Errorf("database connection is not established")
	}
	if len(key) > 1024 {
		key = key[:1024]
	}

	tx, err := db.connx.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if len(tags) == 0 {
		if _, err := tx.Exec(`DELETE FROM tagged_posts WHERE user_account_id = ? AND post_key_hash = ?`, userID, keyHash); err != nil {
			db.Logger().Errorf("Failed to untag post for user %d: %v", userID, err)
			return err
		}
		return tx.Commit()
	}

	if _, err := tx.Exec(`
		INSERT INTO tagged_posts (user_account_id, post_key_hash, post_key, social_account_id, snapshot, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, NOW(3), NOW(3))
		ON DUPLICATE KEY UPDATE social_account_id = VALUES(social_account_id), snapshot = VALUES(snapshot), updated_at = NOW(3)`,
		userID, keyHash, key, socialAccountID, snapshot); err != nil {
		db.Logger().Errorf("Failed to save tagged post for user %d: %v", userID, err)
		return err
	}

	var id uint32
	if err := tx.Get(&id, `SELECT id FROM tagged_posts WHERE user_account_id = ? AND post_key_hash = ?`, userID, keyHash); err != nil {
		return err
	}

	args := []any{id}
	for _, t := range tags {
		args = append(args, t)
	}
	if _, err := tx.Exec(`DELETE FROM post_tags WHERE tagged_post_id = ? AND tag NOT IN (`+placeholders(len(tags))+`)`, args...); err != nil {
		db.Logger().Errorf("Failed to remove post tags: %v", err)
		return err
	}
	// INSERT IGNORE keeps when each existing tag was first added, which orders tag columns.
	for _, t := range tags {
		if _, err := tx.Exec(`INSERT IGNORE INTO post_tags (tagged_post_id, tag, created_at) VALUES (?, ?, NOW(3))`, id, t); err != nil {
			db.Logger().Errorf("Failed to add post tag: %v", err)
			return err
		}
	}
	return tx.Commit()
}

// SelectPostTagsByHash returns a user's tags, keyed by post key hash.
func (db *DB) SelectPostTagsByHash(userID uint32) (map[string][]string, error) {
	out := map[string][]string{}
	rows := make([]struct {
		PostKeyHash string `db:"post_key_hash"`
		Tag         string `db:"tag"`
	}, 0)
	err := db.ResilientSelect(&rows, `
		SELECT tp.post_key_hash, pt.tag FROM post_tags pt
		JOIN tagged_posts tp ON tp.id = pt.tagged_post_id
		WHERE tp.user_account_id = ?
		ORDER BY pt.created_at, pt.tag`, userID)
	if err != nil {
		db.Logger().Errorf("Failed to select post tags for user %d: %v", userID, err)
		return out, err
	}
	for _, r := range rows {
		out[r.PostKeyHash] = append(out[r.PostKeyHash], r.Tag)
	}
	return out, nil
}

// SelectTaggedPosts returns a user's posts carrying tag, most recently tagged first.
func (db *DB) SelectTaggedPosts(userID uint32, tag string, limit int) ([]*TaggedPost, error) {
	out := make([]*TaggedPost, 0)
	err := db.ResilientSelect(&out, `
		SELECT tp.id, tp.post_key_hash, tp.post_key, tp.social_account_id, tp.snapshot, pt.created_at AS tagged_at
		FROM post_tags pt
		JOIN tagged_posts tp ON tp.id = pt.tagged_post_id
		WHERE tp.user_account_id = ? AND pt.tag = ?
		ORDER BY pt.created_at DESC, tp.id DESC
		LIMIT ?`, userID, tag, limit)
	if err != nil {
		db.Logger().Errorf("Failed to select tagged posts for user %d: %v", userID, err)
	}
	return out, err
}

// SelectTagCounts returns every tag a user has used, most used first.
func (db *DB) SelectTagCounts(userID uint32) ([]*TagCount, error) {
	out := make([]*TagCount, 0)
	err := db.ResilientSelect(&out, `
		SELECT pt.tag, COUNT(*) AS posts FROM post_tags pt
		JOIN tagged_posts tp ON tp.id = pt.tagged_post_id
		WHERE tp.user_account_id = ?
		GROUP BY pt.tag
		ORDER BY posts DESC, pt.tag`, userID)
	if err != nil {
		db.Logger().Errorf("Failed to select tags for user %d: %v", userID, err)
	}
	return out, err
}

func placeholders(n int) string {
	return strings.TrimSuffix(strings.Repeat("?,", n), ",")
}
