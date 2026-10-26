ALTER TABLE `feed_columns`
  ADD COLUMN `social_account_id` int unsigned DEFAULT NULL AFTER `user_account_id`;

UPDATE `feed_columns` c
  SET c.`social_account_id` = (
    SELECT MIN(a.`social_account_id`) FROM `feed_column_social_accounts` a WHERE a.`feed_column_id` = c.`id`
  );

DELETE FROM `feed_columns` WHERE `social_account_id` IS NULL;

ALTER TABLE `feed_columns`
  MODIFY `social_account_id` int unsigned NOT NULL,
  ADD CONSTRAINT `feed_columns_social_account_fk`
    FOREIGN KEY (`social_account_id`) REFERENCES `social_accounts` (`id`)
    ON DELETE CASCADE;

DROP TABLE IF EXISTS `feed_column_social_accounts`;
