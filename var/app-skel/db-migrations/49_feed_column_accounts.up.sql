CREATE TABLE `feed_column_social_accounts` (
  `feed_column_id` int unsigned NOT NULL,
  `social_account_id` int unsigned NOT NULL,
  PRIMARY KEY (`feed_column_id`, `social_account_id`),
  KEY `feed_column_social_accounts_account_idx` (`social_account_id`),
  CONSTRAINT `feed_column_social_accounts_column_fk`
    FOREIGN KEY (`feed_column_id`) REFERENCES `feed_columns` (`id`)
    ON DELETE CASCADE,
  CONSTRAINT `feed_column_social_accounts_account_fk`
    FOREIGN KEY (`social_account_id`) REFERENCES `social_accounts` (`id`)
    ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

INSERT INTO `feed_column_social_accounts` (`feed_column_id`, `social_account_id`)
  SELECT `id`, `social_account_id` FROM `feed_columns`;

ALTER TABLE `feed_columns`
  DROP FOREIGN KEY `feed_columns_social_account_fk`,
  DROP COLUMN `social_account_id`;
