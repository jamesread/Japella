CREATE TABLE `rbac_user_roles` (
  `user_account_id` int unsigned NOT NULL,
  `role_id` int unsigned NOT NULL,
  PRIMARY KEY (`user_account_id`,`role_id`),
  KEY `fk_rur_role` (`role_id`),
  CONSTRAINT `fk_rur_user` FOREIGN KEY (`user_account_id`) REFERENCES `user_accounts` (`id`) ON DELETE CASCADE,
  CONSTRAINT `fk_rur_role` FOREIGN KEY (`role_id`) REFERENCES `rbac_roles` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb3 COLLATE=utf8mb3_bin;

DROP TABLE IF EXISTS `rbac_group_roles`;

UPDATE `rbac_permissions`
SET `description` = 'Create, update, and delete roles; assign roles to users'
WHERE `name` = 'rbac.manage';
