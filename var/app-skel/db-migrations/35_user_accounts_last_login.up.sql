ALTER TABLE `user_accounts`
  ADD COLUMN `last_login_at` datetime(3) DEFAULT NULL AFTER `updated_at`;
