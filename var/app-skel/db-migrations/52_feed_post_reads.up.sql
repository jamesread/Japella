CREATE TABLE `feed_post_reads` (
  `user_account_id` int unsigned NOT NULL,
  `feed_id` int NOT NULL, -- feed.id is a signed INT
  `created_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`user_account_id`, `feed_id`),
  KEY `feed_post_reads_feed_idx` (`feed_id`),
  CONSTRAINT `feed_post_reads_user_fk`
    FOREIGN KEY (`user_account_id`) REFERENCES `user_accounts` (`id`)
    ON DELETE CASCADE,
  CONSTRAINT `feed_post_reads_feed_fk`
    FOREIGN KEY (`feed_id`) REFERENCES `feed` (`id`)
    ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
