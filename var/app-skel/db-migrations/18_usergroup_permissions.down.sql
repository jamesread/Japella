DELETE rp FROM `rbac_role_permissions` rp
INNER JOIN `rbac_permissions` p ON rp.permission_id = p.id
WHERE p.name IN ('usergroups.view', 'usergroups.manage');

DELETE FROM `rbac_permissions` WHERE name IN ('usergroups.view', 'usergroups.manage');
