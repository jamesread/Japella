ALTER TABLE `chat_bot_messages`
  DROP KEY `idx_chat_bot_messages_protocol_bot_id`,
  DROP COLUMN `bot_id`;

ALTER TABLE `webhook_hooks`
  DROP KEY `idx_webhook_hooks_protocol_bot_id`,
  DROP COLUMN `bot_id`;

ALTER TABLE `cvars` MODIFY COLUMN `key_name` varchar(64) NOT NULL;

DROP TABLE IF EXISTS `chat_bot_instances`;
