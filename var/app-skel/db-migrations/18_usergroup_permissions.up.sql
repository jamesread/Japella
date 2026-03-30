INSERT INTO `rbac_permissions` (`created_at`, `updated_at`, `name`, `description`) VALUES
(NOW(3), NOW(3), 'usergroups.view', 'List user groups and view membership'),
(NOW(3), NOW(3), 'usergroups.manage', 'Create and delete user groups; manage membership');

INSERT INTO `rbac_role_permissions` (`role_id`, `permission_id`)
SELECT r.id, p.id FROM `rbac_roles` r CROSS JOIN `rbac_permissions` p
WHERE r.name = 'superuser' AND p.name IN ('usergroups.view', 'usergroups.manage');
