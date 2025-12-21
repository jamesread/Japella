ALTER TABLE feed ADD UNIQUE INDEX idx_feed_unique_post (social_account_id, remote_id);

