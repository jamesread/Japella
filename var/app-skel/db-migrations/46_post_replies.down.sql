DROP TABLE IF EXISTS `post_replies`;

ALTER TABLE `post_engagement`
  DROP COLUMN `replies_fetched_at`,
  DROP COLUMN `replies_fetched_count`;
