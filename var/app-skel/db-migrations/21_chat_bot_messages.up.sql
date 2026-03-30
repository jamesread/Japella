CREATE TABLE `chat_bot_messages` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `connector` varchar(50) NOT NULL,
  `identity` varchar(255) NOT NULL,
  `conversation_key` varchar(255) NOT NULL,
  `conversation_title` varchar(255) NOT NULL,
  `channel` varchar(255) NOT NULL,
  `author` varchar(255) NOT NULL,
  `content` text NOT NULL,
  `direction` varchar(20) NOT NULL,
  `message_id` varchar(255) NOT NULL DEFAULT '',
  `timestamp_unix` bigint NOT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_chat_bot_messages_bot` (`connector`, `identity`),
  KEY `idx_chat_bot_messages_conversation` (`connector`, `identity`, `conversation_key`),
  KEY `idx_chat_bot_messages_time` (`timestamp_unix`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb3 COLLATE=utf8mb3_bin;
