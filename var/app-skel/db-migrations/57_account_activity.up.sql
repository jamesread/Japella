-- Likes and comments a social account made through Japella, for its recent
-- interactions. Likes are one row per post (removed on unlike); comments are
-- one row each.
CREATE TABLE `account_activity` (
  `id` int unsigned NOT NULL AUTO_INCREMENT,
  `social_account_id` int unsigned NOT NULL,
  -- "liked" or "commented".
  `kind` varchar(16) NOT NULL,
  -- sha256 of the post key (see feedauthor.PostKey), as in post_comments_made.
  `post_key_hash` char(64) NOT NULL,
  -- Set to post_key_hash for likes so each post is liked once; NULL for comments.
  `like_key` char(64) DEFAULT NULL,
  `post_url` varchar(2048) NOT NULL DEFAULT '',
  `post_author` varchar(255) NOT NULL DEFAULT '',
  `post_snippet` varchar(1024) NOT NULL DEFAULT '',
  `comment_content` text,
  `comment_url` varchar(2048) NOT NULL DEFAULT '',
  `created_at` datetime(3) NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `account_activity_like` (`social_account_id`, `kind`, `like_key`),
  KEY `account_activity_account_time` (`social_account_id`, `created_at`),
  CONSTRAINT `account_activity_account_fk`
    FOREIGN KEY (`social_account_id`) REFERENCES `social_accounts` (`id`)
    ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
