ALTER TABLE `api_keys`
  ADD COLUMN `last_used_at` datetime(3) DEFAULT NULL;
