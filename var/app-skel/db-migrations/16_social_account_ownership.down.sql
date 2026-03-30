DROP TABLE IF EXISTS social_account_shares;
ALTER TABLE social_accounts DROP FOREIGN KEY fk_sa_owner;
ALTER TABLE social_accounts DROP KEY fk_sa_owner;
ALTER TABLE social_accounts DROP COLUMN owner_user_id;

DELETE rp FROM rbac_role_permissions rp
  JOIN rbac_permissions p ON p.id = rp.permission_id
  WHERE p.name IN ('social-accounts.view-all', 'system.settings', 'system.connectors', 'system.logs');
DELETE FROM rbac_permissions WHERE name IN ('social-accounts.view-all', 'system.settings', 'system.connectors', 'system.logs');
