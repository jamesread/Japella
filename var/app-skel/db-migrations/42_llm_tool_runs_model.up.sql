ALTER TABLE `llm_tool_runs`
  ADD COLUMN `model` varchar(255) NOT NULL DEFAULT '' AFTER `connection_name`;
