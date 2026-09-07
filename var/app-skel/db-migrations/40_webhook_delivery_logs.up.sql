CREATE TABLE `webhook_delivery_logs` (
  `id` int NOT NULL AUTO_INCREMENT,
  `webhook_target_id` int NOT NULL,
  `event` varchar(64) NOT NULL,
  `success` tinyint NOT NULL DEFAULT 0,
  `http_status` int NOT NULL DEFAULT 0,
  `error_message` varchar(1024) NOT NULL DEFAULT '',
  `duration_ms` int NOT NULL DEFAULT 0,
  `created_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `webhook_delivery_logs_target_idx` (`webhook_target_id`),
  KEY `webhook_delivery_logs_created_idx` (`created_at`),
  CONSTRAINT `webhook_delivery_logs_target_fk`
    FOREIGN KEY (`webhook_target_id`) REFERENCES `webhook_targets` (`id`)
    ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
