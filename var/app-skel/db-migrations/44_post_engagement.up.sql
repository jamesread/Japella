CREATE TABLE `post_engagement` (
  `post_id` int(10) unsigned NOT NULL,
  `likes` int unsigned NOT NULL DEFAULT 0,
  `reposts` int unsigned NOT NULL DEFAULT 0,
  `replies` int unsigned NOT NULL DEFAULT 0,
  `quotes` int unsigned NOT NULL DEFAULT 0,
  `impressions` int unsigned NOT NULL DEFAULT 0,
  `last_fetched_at` datetime(3) DEFAULT NULL,
  `last_changed_at` datetime(3) DEFAULT NULL,
  `last_error` varchar(1024) NOT NULL DEFAULT '',
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`post_id`),
  KEY `post_engagement_last_fetched_idx` (`last_fetched_at`),
  CONSTRAINT `post_engagement_post_fk`
    FOREIGN KEY (`post_id`) REFERENCES `posts` (`id`)
    ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
