ALTER TABLE `feed`
  ADD COLUMN `author_handle` varchar(255) NOT NULL DEFAULT '',
  ADD COLUMN `author_profile_url` varchar(1024) NOT NULL DEFAULT '',
  ADD COLUMN `original_author_handle` varchar(255) NOT NULL DEFAULT '',
  ADD COLUMN `original_author_profile_url` varchar(1024) NOT NULL DEFAULT '';
