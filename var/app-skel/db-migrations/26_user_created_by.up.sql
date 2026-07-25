ALTER TABLE `user_accounts`
  ADD COLUMN `created_by` varchar(64) NOT NULL DEFAULT 'admin-created';
