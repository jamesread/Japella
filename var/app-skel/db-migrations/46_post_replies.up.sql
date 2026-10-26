ALTER TABLE `post_engagement`
  ADD COLUMN `replies_fetched_at` datetime(3) DEFAULT NULL,
  ADD COLUMN `replies_fetched_count` int unsigned NOT NULL DEFAULT 0;

CREATE TABLE `post_replies` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `post_id` int(10) unsigned NOT NULL,
  `remote_id` varchar(512) NOT NULL,
  `parent_remote_id` varchar(512) NOT NULL DEFAULT '',
  `remote_url` varchar(1024) NOT NULL DEFAULT '',
  `author_name` varchar(255) NOT NULL DEFAULT '',
  `author_handle` varchar(255) NOT NULL DEFAULT '',
  `author_avatar_url` varchar(1024) NOT NULL DEFAULT '',
  `content` text NOT NULL,
  `posted_at` datetime(3) DEFAULT NULL,
  `created_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `post_replies_post_idx` (`post_id`),
  CONSTRAINT `post_replies_post_fk`
    FOREIGN KEY (`post_id`) REFERENCES `posts` (`id`)
    ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
