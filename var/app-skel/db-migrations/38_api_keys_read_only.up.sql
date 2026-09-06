ALTER TABLE `api_keys`
  ADD COLUMN `read_only` tinyint(1) NOT NULL DEFAULT 0 AFTER `name`;
