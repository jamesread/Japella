package db

// MarkFeedPostRead hides a feed post from one user's feed.
func (db *DB) MarkFeedPostRead(userID, feedID uint32) error {
	_, err := db.ResilientExec(`
		INSERT IGNORE INTO feed_post_reads (user_account_id, feed_id, created_at)
		VALUES (?, ?, NOW(3))`, userID, feedID)
	if err != nil {
		db.Logger().Errorf("Failed to mark feed post %d read for user %d: %v", feedID, userID, err)
	}
	return err
}

// SelectReadFeedPostIDs returns the feed posts a user has marked read.
func (db *DB) SelectReadFeedPostIDs(userID uint32) (map[uint32]bool, error) {
	ids := make([]uint32, 0)
	if err := db.ResilientSelect(&ids, `SELECT feed_id FROM feed_post_reads WHERE user_account_id = ?`, userID); err != nil {
		db.Logger().Errorf("Failed to select read feed posts for user %d: %v", userID, err)
		return nil, err
	}
	out := make(map[uint32]bool, len(ids))
	for _, id := range ids {
		out[id] = true
	}
	return out, nil
}

// MarkRemotePostRead hides a post from one user's feed and feed columns,
// identified by its platform key (see feedauthor.PostKey).
func (db *DB) MarkRemotePostRead(userID uint32, keyHash, key string) error {
	if len(key) > 1024 {
		key = key[:1024]
	}
	_, err := db.ResilientExec(`
		INSERT IGNORE INTO remote_post_reads (user_account_id, post_key_hash, post_key, created_at)
		VALUES (?, ?, ?, NOW(3))`, userID, keyHash, key)
	if err != nil {
		db.Logger().Errorf("Failed to mark remote post read for user %d: %v", userID, err)
	}
	return err
}

// SelectReadRemotePostHashes returns the key hashes of posts a user marked read.
func (db *DB) SelectReadRemotePostHashes(userID uint32) (map[string]bool, error) {
	hashes := make([]string, 0)
	if err := db.ResilientSelect(&hashes, `SELECT post_key_hash FROM remote_post_reads WHERE user_account_id = ?`, userID); err != nil {
		db.Logger().Errorf("Failed to select read remote posts for user %d: %v", userID, err)
		return nil, err
	}
	out := make(map[string]bool, len(hashes))
	for _, h := range hashes {
		out[h] = true
	}
	return out, nil
}
