CREATE TABLE `post_comments_made` (
  `social_account_id` int unsigned NOT NULL,
  -- sha256 of the post's remote URL (or remote id), as in remote_post_reads.
  `post_key_hash` char(64) NOT NULL,
  `post_key` varchar(1024) NOT NULL,
  `created_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`social_account_id`, `post_key_hash`),
  CONSTRAINT `post_comments_made_account_fk`
    FOREIGN KEY (`social_account_id`) REFERENCES `social_accounts` (`id`)
    ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
