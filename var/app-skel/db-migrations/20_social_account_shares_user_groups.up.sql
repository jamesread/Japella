-- Replace per-user social_account_shares with per-user-group shares.
-- Existing rows referred to user_account_id and cannot be mapped automatically; they are dropped.

DROP TABLE IF EXISTS `social_account_shares`;

CREATE TABLE `social_account_shares` (
  `id` INT UNSIGNED NOT NULL AUTO_INCREMENT,
  `social_account_id` INT UNSIGNED NOT NULL,
  `user_group_id` INT UNSIGNED NOT NULL,
  `can_read` TINYINT(1) NOT NULL DEFAULT 1,
  `can_post` TINYINT(1) NOT NULL DEFAULT 0,
  `can_manage` TINYINT(1) NOT NULL DEFAULT 0,
  `created_at` DATETIME(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_sa_share` (`social_account_id`, `user_group_id`),
  KEY `fk_sas_ug` (`user_group_id`),
  CONSTRAINT `fk_sas_sa` FOREIGN KEY (`social_account_id`)
    REFERENCES `social_accounts` (`id`) ON DELETE CASCADE,
  CONSTRAINT `fk_sas_ug` FOREIGN KEY (`user_group_id`)
    REFERENCES `user_groups` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb3 COLLATE=utf8mb3_bin;
