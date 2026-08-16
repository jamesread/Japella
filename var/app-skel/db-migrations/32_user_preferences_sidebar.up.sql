ALTER TABLE `user_preferences`
  ADD COLUMN `sidebar_enabled` tinyint(1) NOT NULL DEFAULT 1 AFTER `language`;
