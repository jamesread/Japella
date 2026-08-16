CREATE TABLE `user_preferences` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `user_account_id` int(10) unsigned NOT NULL,
  `language` varchar(32) NOT NULL DEFAULT '',
  `sidebar_enabled` tinyint(1) NOT NULL DEFAULT 1,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_user_preferences_user_account_id` (`user_account_id`),
  CONSTRAINT `fk_user_preferences_user_account` FOREIGN KEY (`user_account_id`) REFERENCES `user_accounts` (`id`) ON DELETE CASCADE ON UPDATE NO ACTION
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb3 COLLATE=utf8mb3_bin;
