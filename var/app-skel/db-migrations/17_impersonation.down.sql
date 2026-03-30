ALTER TABLE sessions DROP FOREIGN KEY fk_sessions_impersonator;
ALTER TABLE sessions DROP COLUMN impersonator_user_id;

DELETE rp FROM rbac_role_permissions rp
  JOIN rbac_permissions p ON p.id = rp.permission_id
  WHERE p.name = 'system.impersonate';
DELETE FROM rbac_permissions WHERE name = 'system.impersonate';
