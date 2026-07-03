CREATE TABLE `chat_bot_instances` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `protocol` varchar(32) NOT NULL,
  `bot_id` varchar(32) NOT NULL,
  `display_name` varchar(255) NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_chat_bot_instances_protocol_bot_id` (`protocol`, `bot_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb3 COLLATE=utf8mb3_bin;

ALTER TABLE `cvars` MODIFY COLUMN `key_name` varchar(128) NOT NULL;

ALTER TABLE `webhook_hooks`
  ADD COLUMN `bot_id` varchar(32) NOT NULL DEFAULT '' AFTER `identity`,
  ADD KEY `idx_webhook_hooks_protocol_bot_id` (`connector`, `bot_id`);

ALTER TABLE `chat_bot_messages`
  ADD COLUMN `bot_id` varchar(32) NOT NULL DEFAULT '' AFTER `identity`,
  ADD KEY `idx_chat_bot_messages_protocol_bot_id` (`connector`, `bot_id`);
