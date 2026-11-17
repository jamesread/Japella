package db

import "strings"

// RecordPostCommented notes that a social account has commented on a post,
// identified by its platform key (see feedauthor.PostKey).
func (db *DB) RecordPostCommented(socialAccountID uint32, keyHash, key string) error {
	if len(key) > 1024 {
		key = key[:1024]
	}
	_, err := db.ResilientExec(`
		INSERT IGNORE INTO post_comments_made (social_account_id, post_key_hash, post_key, created_at)
		VALUES (?, ?, ?, NOW(3))`, socialAccountID, keyHash, key)
	if err != nil {
		db.Logger().Errorf("Failed to record comment by social account %d: %v", socialAccountID, err)
	}
	return err
}

func uint32Args(ids []uint32) []any {
	args := make([]any, len(ids))
	for i, id := range ids {
		args[i] = id
	}
	return args
}

func addCommentedHash(out map[uint32]map[string]bool, socialAccountID uint32, hash string) {
	if out[socialAccountID] == nil {
		out[socialAccountID] = map[string]bool{}
	}
	out[socialAccountID][hash] = true
}

// SelectCommentedPostHashes returns, per social account, the key hashes of
// posts it has commented on.
func (db *DB) SelectCommentedPostHashes(socialAccountIDs []uint32) (map[uint32]map[string]bool, error) {
	out := make(map[uint32]map[string]bool, len(socialAccountIDs))
	if len(socialAccountIDs) == 0 {
		return out, nil
	}

	args := uint32Args(socialAccountIDs)
	rows := make([]struct {
		SocialAccountID uint32 `db:"social_account_id"`
		PostKeyHash     string `db:"post_key_hash"`
	}, 0)
	err := db.ResilientSelect(&rows, `
		SELECT social_account_id, post_key_hash FROM post_comments_made
		WHERE social_account_id IN (`+strings.TrimSuffix(strings.Repeat("?,", len(args)), ",")+`)`, args...)
	if err != nil {
		db.Logger().Errorf("Failed to select commented posts: %v", err)
		return out, err
	}
	for _, r := range rows {
		addCommentedHash(out, r.SocialAccountID, r.PostKeyHash)
	}
	return out, nil
}
