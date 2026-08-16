CREATE TABLE `rbac_group_roles` (
  `user_group_id` int unsigned NOT NULL,
  `role_id` int unsigned NOT NULL,
  PRIMARY KEY (`user_group_id`,`role_id`),
  KEY `fk_rgr_role` (`role_id`),
  CONSTRAINT `fk_rgr_group` FOREIGN KEY (`user_group_id`) REFERENCES `user_groups` (`id`) ON DELETE CASCADE,
  CONSTRAINT `fk_rgr_role` FOREIGN KEY (`role_id`) REFERENCES `rbac_roles` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb3 COLLATE=utf8mb3_bin;

INSERT INTO `user_groups` (`name`, `created_at`, `updated_at`)
SELECT 'Everyone', NOW(3), NOW(3) FROM DUAL
WHERE NOT EXISTS (SELECT 1 FROM `user_groups` WHERE `name` = 'Everyone');

INSERT INTO `user_groups` (`name`, `created_at`, `updated_at`)
SELECT 'Administrators', NOW(3), NOW(3) FROM DUAL
WHERE NOT EXISTS (SELECT 1 FROM `user_groups` WHERE `name` = 'Administrators');

INSERT IGNORE INTO `rbac_group_roles` (`user_group_id`, `role_id`)
SELECT g.id, r.id FROM `user_groups` g
CROSS JOIN `rbac_roles` r
WHERE g.name = 'Everyone' AND r.name = 'member';

INSERT IGNORE INTO `rbac_group_roles` (`user_group_id`, `role_id`)
SELECT g.id, r.id FROM `user_groups` g
CROSS JOIN `rbac_roles` r
WHERE g.name = 'Administrators' AND r.name = 'superuser';

INSERT IGNORE INTO `user_group_memberships` (`user_account_id`, `user_group_id`, `created_at`, `updated_at`)
SELECT u.id, g.id, NOW(3), NOW(3)
FROM `user_accounts` u
CROSS JOIN `user_groups` g
WHERE g.name = 'Everyone';

INSERT IGNORE INTO `user_group_memberships` (`user_account_id`, `user_group_id`, `created_at`, `updated_at`)
SELECT ur.user_account_id, g.id, NOW(3), NOW(3)
FROM `rbac_user_roles` ur
INNER JOIN `rbac_roles` r ON r.id = ur.role_id AND r.name = 'superuser'
CROSS JOIN `user_groups` g
WHERE g.name = 'Administrators';

INSERT INTO `user_groups` (`name`, `created_at`, `updated_at`)
SELECT CONCAT('Role: ', r.name), NOW(3), NOW(3)
FROM `rbac_roles` r
WHERE r.name NOT IN ('superuser', 'member')
AND EXISTS (SELECT 1 FROM `rbac_user_roles` ur WHERE ur.role_id = r.id)
AND NOT EXISTS (SELECT 1 FROM `user_groups` g WHERE g.name = CONCAT('Role: ', r.name));

INSERT IGNORE INTO `rbac_group_roles` (`user_group_id`, `role_id`)
SELECT g.id, r.id
FROM `rbac_roles` r
INNER JOIN `user_groups` g ON g.name = CONCAT('Role: ', r.name)
WHERE r.name NOT IN ('superuser', 'member');

INSERT IGNORE INTO `user_group_memberships` (`user_account_id`, `user_group_id`, `created_at`, `updated_at`)
SELECT ur.user_account_id, g.id, NOW(3), NOW(3)
FROM `rbac_user_roles` ur
INNER JOIN `rbac_roles` r ON r.id = ur.role_id
INNER JOIN `user_groups` g ON g.name = CONCAT('Role: ', r.name)
WHERE r.name NOT IN ('superuser', 'member');

UPDATE `rbac_permissions`
SET `description` = 'Create, update, and delete roles; assign roles to groups'
WHERE `name` = 'rbac.manage';

DROP TABLE `rbac_user_roles`;
