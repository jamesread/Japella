CREATE TABLE `account_policies` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `name` varchar(191) NOT NULL,
  `description` varchar(512) NOT NULL DEFAULT '',
  `apply_to_mcp` tinyint(1) NOT NULL DEFAULT 0,
  `apply_to_ui` tinyint(1) NOT NULL DEFAULT 0,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_account_policies_name` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb3 COLLATE=utf8mb3_bin;

CREATE TABLE `account_policy_social_accounts` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `account_policy_id` int(10) unsigned NOT NULL,
  `social_account_id` int(10) unsigned NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_apsa_social_account` (`social_account_id`),
  KEY `fk_apsa_policy` (`account_policy_id`),
  CONSTRAINT `fk_apsa_policy` FOREIGN KEY (`account_policy_id`) REFERENCES `account_policies` (`id`) ON DELETE CASCADE,
  CONSTRAINT `fk_apsa_social_account` FOREIGN KEY (`social_account_id`) REFERENCES `social_accounts` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb3 COLLATE=utf8mb3_bin;

CREATE TABLE `account_policy_approval_stages` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `account_policy_id` int(10) unsigned NOT NULL,
  `stage_order` int unsigned NOT NULL DEFAULT 0,
  `user_id` int(10) unsigned DEFAULT NULL,
  `user_group_id` int(10) unsigned DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_apas_policy_order` (`account_policy_id`, `stage_order`),
  KEY `fk_apas_user` (`user_id`),
  KEY `fk_apas_user_group` (`user_group_id`),
  CONSTRAINT `fk_apas_policy` FOREIGN KEY (`account_policy_id`) REFERENCES `account_policies` (`id`) ON DELETE CASCADE,
  CONSTRAINT `fk_apas_user` FOREIGN KEY (`user_id`) REFERENCES `user_accounts` (`id`) ON DELETE CASCADE,
  CONSTRAINT `fk_apas_user_group` FOREIGN KEY (`user_group_id`) REFERENCES `user_groups` (`id`) ON DELETE CASCADE,
  CONSTRAINT `chk_apas_assignee` CHECK (
    (`user_id` IS NOT NULL AND `user_group_id` IS NULL) OR
    (`user_id` IS NULL AND `user_group_id` IS NOT NULL)
  )
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb3 COLLATE=utf8mb3_bin;

CREATE TABLE `post_approvals` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `post_id` int(10) unsigned NOT NULL,
  `stage_id` int(10) unsigned NOT NULL,
  `approved_by_user_id` int(10) unsigned NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_post_approvals_post_stage` (`post_id`, `stage_id`),
  KEY `fk_pa_stage` (`stage_id`),
  KEY `fk_pa_user` (`approved_by_user_id`),
  CONSTRAINT `fk_pa_post` FOREIGN KEY (`post_id`) REFERENCES `posts` (`id`) ON DELETE CASCADE,
  CONSTRAINT `fk_pa_stage` FOREIGN KEY (`stage_id`) REFERENCES `account_policy_approval_stages` (`id`) ON DELETE CASCADE,
  CONSTRAINT `fk_pa_user` FOREIGN KEY (`approved_by_user_id`) REFERENCES `user_accounts` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb3 COLLATE=utf8mb3_bin;

ALTER TABLE `posts`
  ADD COLUMN `submission_source` varchar(16) NOT NULL DEFAULT 'ui' AFTER `campaign_id`,
  ADD COLUMN `submitted_by_user_id` int(10) unsigned DEFAULT NULL AFTER `submission_source`,
  ADD COLUMN `account_policy_id` int(10) unsigned DEFAULT NULL AFTER `submitted_by_user_id`,
  ADD COLUMN `approval_stage` int unsigned NOT NULL DEFAULT 0 AFTER `account_policy_id`,
  ADD KEY `fk_posts_submitted_by` (`submitted_by_user_id`),
  ADD KEY `fk_posts_account_policy` (`account_policy_id`),
  ADD CONSTRAINT `fk_posts_submitted_by` FOREIGN KEY (`submitted_by_user_id`) REFERENCES `user_accounts` (`id`) ON DELETE SET NULL,
  ADD CONSTRAINT `fk_posts_account_policy` FOREIGN KEY (`account_policy_id`) REFERENCES `account_policies` (`id`) ON DELETE SET NULL;

INSERT INTO `rbac_permissions` (`created_at`, `updated_at`, `name`, `description`) VALUES
(NOW(3), NOW(3), 'account-policies.manage', 'Create and manage account policies and approval stages');

INSERT INTO `rbac_role_permissions` (`role_id`, `permission_id`)
SELECT r.id, p.id FROM `rbac_roles` r CROSS JOIN `rbac_permissions` p
WHERE r.name = 'superuser' AND p.name = 'account-policies.manage';
