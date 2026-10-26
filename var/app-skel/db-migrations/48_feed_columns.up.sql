CREATE TABLE `feed_columns` (
  `id` int unsigned NOT NULL AUTO_INCREMENT,
  `user_account_id` int unsigned NOT NULL,
  `social_account_id` int unsigned NOT NULL,
  `title` varchar(255) NOT NULL DEFAULT '',
  `query` varchar(255) NOT NULL,
  `sort_order` int unsigned NOT NULL DEFAULT 0,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `feed_columns_user_idx` (`user_account_id`, `sort_order`),
  CONSTRAINT `feed_columns_user_fk`
    FOREIGN KEY (`user_account_id`) REFERENCES `user_accounts` (`id`)
    ON DELETE CASCADE,
  CONSTRAINT `feed_columns_social_account_fk`
    FOREIGN KEY (`social_account_id`) REFERENCES `social_accounts` (`id`)
    ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
