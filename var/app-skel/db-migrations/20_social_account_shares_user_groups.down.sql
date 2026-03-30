DROP TABLE IF EXISTS `social_account_shares`;

CREATE TABLE `social_account_shares` (
  `id` INT UNSIGNED NOT NULL AUTO_INCREMENT,
  `social_account_id` INT UNSIGNED NOT NULL,
  `user_account_id` INT UNSIGNED NOT NULL,
  `can_read` TINYINT(1) NOT NULL DEFAULT 1,
  `can_post` TINYINT(1) NOT NULL DEFAULT 0,
  `can_manage` TINYINT(1) NOT NULL DEFAULT 0,
  `created_at` DATETIME(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_sa_share` (`social_account_id`, `user_account_id`),
  CONSTRAINT `fk_sas_sa` FOREIGN KEY (`social_account_id`)
    REFERENCES `social_accounts` (`id`) ON DELETE CASCADE,
  CONSTRAINT `fk_sas_user` FOREIGN KEY (`user_account_id`)
    REFERENCES `user_accounts` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb3 COLLATE=utf8mb3_bin;
