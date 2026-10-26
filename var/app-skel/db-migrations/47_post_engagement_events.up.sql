CREATE TABLE `post_engagement_events` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `post_id` int(10) unsigned NOT NULL,
  `social_account_id` int(10) unsigned NOT NULL,
  `likes` int unsigned NOT NULL DEFAULT 0,
  `reposts` int unsigned NOT NULL DEFAULT 0,
  `replies` int unsigned NOT NULL DEFAULT 0,
  `quotes` int unsigned NOT NULL DEFAULT 0,
  `detected_at` datetime(3) NOT NULL,
  PRIMARY KEY (`id`),
  KEY `post_engagement_events_account_idx` (`social_account_id`, `detected_at`),
  KEY `post_engagement_events_detected_idx` (`detected_at`),
  CONSTRAINT `post_engagement_events_post_fk`
    FOREIGN KEY (`post_id`) REFERENCES `posts` (`id`)
    ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
