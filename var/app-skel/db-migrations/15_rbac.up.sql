CREATE TABLE `rbac_permissions` (
  `id` int unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `name` varchar(191) NOT NULL,
  `description` varchar(512) NOT NULL DEFAULT '',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_rbac_permissions_name` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb3 COLLATE=utf8mb3_bin;

CREATE TABLE `rbac_roles` (
  `id` int unsigned NOT NULL AUTO_INCREMENT,
  `created_at` datetime(3) DEFAULT NULL,
  `updated_at` datetime(3) DEFAULT NULL,
  `name` varchar(191) NOT NULL,
  `description` varchar(512) NOT NULL DEFAULT '',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_rbac_roles_name` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb3 COLLATE=utf8mb3_bin;

CREATE TABLE `rbac_role_permissions` (
  `role_id` int unsigned NOT NULL,
  `permission_id` int unsigned NOT NULL,
  PRIMARY KEY (`role_id`,`permission_id`),
  KEY `fk_rbp_perm` (`permission_id`),
  CONSTRAINT `fk_rbp_role` FOREIGN KEY (`role_id`) REFERENCES `rbac_roles` (`id`) ON DELETE CASCADE,
  CONSTRAINT `fk_rbp_perm` FOREIGN KEY (`permission_id`) REFERENCES `rbac_permissions` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb3 COLLATE=utf8mb3_bin;

CREATE TABLE `rbac_user_roles` (
  `user_account_id` int unsigned NOT NULL,
  `role_id` int unsigned NOT NULL,
  PRIMARY KEY (`user_account_id`,`role_id`),
  KEY `fk_rur_role` (`role_id`),
  CONSTRAINT `fk_rur_user` FOREIGN KEY (`user_account_id`) REFERENCES `user_accounts` (`id`) ON DELETE CASCADE,
  CONSTRAINT `fk_rur_role` FOREIGN KEY (`role_id`) REFERENCES `rbac_roles` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb3 COLLATE=utf8mb3_bin;

INSERT INTO `rbac_permissions` (`created_at`, `updated_at`, `name`, `description`) VALUES
(NOW(3), NOW(3), 'app.access', 'Use the application (non-admin APIs)'),
(NOW(3), NOW(3), 'users.view', 'List and view user accounts'),
(NOW(3), NOW(3), 'users.create', 'Create user accounts'),
(NOW(3), NOW(3), 'users.delete', 'Delete user accounts'),
(NOW(3), NOW(3), 'users.reset-password', 'Reset passwords for other users'),
(NOW(3), NOW(3), 'system.settings', 'View and modify system settings (cvars)'),
(NOW(3), NOW(3), 'system.connectors', 'View connectors and manage OAuth flows'),
(NOW(3), NOW(3), 'system.logs', 'View logs, job status, and run maintenance tasks'),
(NOW(3), NOW(3), 'system.impersonate', 'Impersonate other users to debug permission issues'),
(NOW(3), NOW(3), 'rbac.view', 'View roles and permissions'),
(NOW(3), NOW(3), 'rbac.manage', 'Create, update, and delete roles; assign roles to users');

INSERT INTO `rbac_roles` (`created_at`, `updated_at`, `name`, `description`) VALUES
(NOW(3), NOW(3), 'superuser', 'All permissions (system role)'),
(NOW(3), NOW(3), 'member', 'Standard application access');

INSERT INTO `rbac_role_permissions` (`role_id`, `permission_id`)
SELECT r.id, p.id FROM `rbac_roles` r CROSS JOIN `rbac_permissions` p WHERE r.name = 'superuser';

INSERT INTO `rbac_role_permissions` (`role_id`, `permission_id`)
SELECT r.id, p.id FROM `rbac_roles` r JOIN `rbac_permissions` p ON p.name = 'app.access' WHERE r.name = 'member';
