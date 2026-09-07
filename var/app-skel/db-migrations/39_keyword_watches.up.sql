CREATE TABLE `keyword_watches` (
  `id` int unsigned NOT NULL AUTO_INCREMENT,
  `social_account_id` int unsigned NOT NULL,
  `keyword` varchar(255) NOT NULL,
  `keyword_normalized` varchar(255) NOT NULL,
  `enabled` tinyint NOT NULL DEFAULT 1,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `keyword_watches_account_keyword_uidx` (`social_account_id`, `keyword_normalized`),
  KEY `keyword_watches_account_idx` (`social_account_id`),
  CONSTRAINT `keyword_watches_social_account_fk`
    FOREIGN KEY (`social_account_id`) REFERENCES `social_accounts` (`id`)
    ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `keyword_watch_state` (
  `social_account_id` int unsigned NOT NULL,
  `acquisition_mode` varchar(32) NOT NULL DEFAULT 'evaluating',
  `native_fidelity` varchar(32) NOT NULL DEFAULT 'search_only',
  `backoff_until` datetime(3) DEFAULT NULL,
  `last_success_at` datetime(3) DEFAULT NULL,
  `last_error_at` datetime(3) DEFAULT NULL,
  `last_error` varchar(1024) NOT NULL DEFAULT '',
  `last_poll_at` datetime(3) DEFAULT NULL,
  `emits_this_hour` int NOT NULL DEFAULT 0,
  `emits_hour_bucket` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`social_account_id`),
  CONSTRAINT `keyword_watch_state_social_account_fk`
    FOREIGN KEY (`social_account_id`) REFERENCES `social_accounts` (`id`)
    ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `keyword_matches` (
  `id` int unsigned NOT NULL AUTO_INCREMENT,
  `social_account_id` int unsigned NOT NULL,
  `keyword_watch_id` int unsigned NOT NULL,
  `connector` varchar(64) NOT NULL,
  `remote_id` varchar(512) NOT NULL,
  `remote_url` varchar(1024) NOT NULL DEFAULT '',
  `author_id` varchar(255) NOT NULL DEFAULT '',
  `author_handle` varchar(255) NOT NULL DEFAULT '',
  `content_snippet` varchar(2048) NOT NULL DEFAULT '',
  `matched_at` datetime(3) NOT NULL,
  `posted_at` datetime(3) DEFAULT NULL,
  `acquisition_mode` varchar(32) NOT NULL DEFAULT 'search',
  `webhook_dispatched_at` datetime(3) DEFAULT NULL,
  `created_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `keyword_matches_watch_remote_uidx` (`keyword_watch_id`, `remote_id`),
  KEY `keyword_matches_account_idx` (`social_account_id`),
  KEY `keyword_matches_matched_at_idx` (`matched_at`),
  CONSTRAINT `keyword_matches_watch_fk`
    FOREIGN KEY (`keyword_watch_id`) REFERENCES `keyword_watches` (`id`)
    ON DELETE CASCADE,
  CONSTRAINT `keyword_matches_social_account_fk`
    FOREIGN KEY (`social_account_id`) REFERENCES `social_accounts` (`id`)
    ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
