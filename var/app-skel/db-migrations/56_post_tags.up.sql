-- Internal, per-user tags on posts from the feed or feed columns. Nothing
-- here is sent to any social network.
CREATE TABLE `tagged_posts` (
  `id` int unsigned NOT NULL AUTO_INCREMENT,
  `user_account_id` int unsigned NOT NULL,
  -- sha256 of the post's remote URL (or remote id), as in remote_post_reads.
  `post_key_hash` char(64) NOT NULL,
  `post_key` varchar(1024) NOT NULL,
  -- The account the post was seen through, used to like/comment from a tag column.
  `social_account_id` int unsigned NOT NULL,
  -- JSON copy of the post as it was when tagged, so tag columns can show
  -- posts that are no longer in the feed or a search.
  `snapshot` mediumtext NOT NULL,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `tagged_posts_user_post_idx` (`user_account_id`, `post_key_hash`),
  CONSTRAINT `tagged_posts_user_fk`
    FOREIGN KEY (`user_account_id`) REFERENCES `user_accounts` (`id`)
    ON DELETE CASCADE,
  CONSTRAINT `tagged_posts_social_account_fk`
    FOREIGN KEY (`social_account_id`) REFERENCES `social_accounts` (`id`)
    ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `post_tags` (
  `tagged_post_id` int unsigned NOT NULL,
  `tag` varchar(64) NOT NULL,
  `created_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`tagged_post_id`, `tag`),
  KEY `post_tags_tag_idx` (`tag`, `created_at`),
  CONSTRAINT `post_tags_tagged_post_fk`
    FOREIGN KEY (`tagged_post_id`) REFERENCES `tagged_posts` (`id`)
    ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
