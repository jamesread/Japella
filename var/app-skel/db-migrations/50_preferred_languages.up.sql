ALTER TABLE `social_accounts`
  ADD COLUMN `preferred_languages` varchar(255) NOT NULL DEFAULT '';

ALTER TABLE `feed`
  ADD COLUMN `language` varchar(16) NOT NULL DEFAULT '';
