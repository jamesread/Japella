CREATE TABLE `webhook_targets` (
  `id` int NOT NULL AUTO_INCREMENT,
  `url` varchar(2048) NOT NULL,
  `secret` varchar(255) NOT NULL,
  `enabled` tinyint NOT NULL DEFAULT 1,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `webhook_events` (
  `id` int NOT NULL AUTO_INCREMENT,
  `webhook_target_id` int NOT NULL,
  `event` varchar(64) NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `webhook_events_target_event_uidx` (`webhook_target_id`, `event`),
  KEY `webhook_events_event_idx` (`event`),
  CONSTRAINT `webhook_events_target_fk`
    FOREIGN KEY (`webhook_target_id`) REFERENCES `webhook_targets` (`id`)
    ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
