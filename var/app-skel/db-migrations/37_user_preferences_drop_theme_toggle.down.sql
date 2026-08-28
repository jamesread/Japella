ALTER TABLE `user_preferences`
  ADD COLUMN `theme_toggle_enabled` tinyint(1) NOT NULL DEFAULT 0 AFTER `sidebar_enabled`;
