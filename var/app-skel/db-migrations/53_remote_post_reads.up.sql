CREATE TABLE `remote_post_reads` (
  `user_account_id` int unsigned NOT NULL,
  -- sha256 of the post's remote URL (or remote id); URLs are too long to index.
  `post_key_hash` char(64) NOT NULL,
  `post_key` varchar(1024) NOT NULL,
  `created_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`user_account_id`, `post_key_hash`),
  CONSTRAINT `remote_post_reads_user_fk`
    FOREIGN KEY (`user_account_id`) REFERENCES `user_accounts` (`id`)
    ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
