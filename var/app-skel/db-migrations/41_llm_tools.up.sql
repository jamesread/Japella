CREATE TABLE `llm_connections` (
  `id` int unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(255) NOT NULL,
  `base_url` varchar(2048) NOT NULL,
  `api_key` varchar(2048) NOT NULL DEFAULT '',
  `model` varchar(255) NOT NULL,
  `enabled` tinyint NOT NULL DEFAULT 1,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `llm_connections_name_uidx` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `llm_tools` (
  `id` int unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(255) NOT NULL,
  `description` varchar(1024) NOT NULL DEFAULT '',
  `llm_connection_id` int unsigned NOT NULL,
  `prompt` text NOT NULL,
  `enabled` tinyint NOT NULL DEFAULT 1,
  `sort_order` int NOT NULL DEFAULT 0,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `llm_tools_name_uidx` (`name`),
  KEY `llm_tools_connection_idx` (`llm_connection_id`),
  CONSTRAINT `llm_tools_connection_fk`
    FOREIGN KEY (`llm_connection_id`) REFERENCES `llm_connections` (`id`)
    ON DELETE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `llm_tool_runs` (
  `id` int unsigned NOT NULL AUTO_INCREMENT,
  `post_id` int unsigned NOT NULL,
  `llm_tool_id` int unsigned DEFAULT NULL,
  `tool_name` varchar(255) NOT NULL,
  `connection_name` varchar(255) NOT NULL DEFAULT '',
  `input_content` text NOT NULL,
  `output_content` text NOT NULL,
  `status` varchar(32) NOT NULL,
  `error_message` varchar(2048) NOT NULL DEFAULT '',
  `created_by_user_id` int unsigned DEFAULT NULL,
  `created_at` datetime(3) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `llm_tool_runs_post_idx` (`post_id`),
  KEY `llm_tool_runs_tool_idx` (`llm_tool_id`),
  CONSTRAINT `llm_tool_runs_post_fk`
    FOREIGN KEY (`post_id`) REFERENCES `posts` (`id`)
    ON DELETE CASCADE,
  CONSTRAINT `llm_tool_runs_tool_fk`
    FOREIGN KEY (`llm_tool_id`) REFERENCES `llm_tools` (`id`)
    ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
