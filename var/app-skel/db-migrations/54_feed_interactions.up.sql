ALTER TABLE `feed`
  ADD COLUMN `interaction_remote_id` varchar(255) NOT NULL DEFAULT '',
  ADD COLUMN `liked` tinyint NOT NULL DEFAULT 0,
  ADD COLUMN `like_count` int unsigned NOT NULL DEFAULT 0,
  ADD COLUMN `reply_count` int unsigned NOT NULL DEFAULT 0;
