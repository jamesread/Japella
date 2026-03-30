ALTER TABLE social_accounts
  ADD COLUMN owner_user_id INT UNSIGNED DEFAULT NULL,
  ADD KEY fk_sa_owner (owner_user_id),
  ADD CONSTRAINT fk_sa_owner FOREIGN KEY (owner_user_id)
    REFERENCES user_accounts(id) ON DELETE SET NULL;

UPDATE social_accounts SET owner_user_id = (SELECT MIN(id) FROM user_accounts)
  WHERE owner_user_id IS NULL;

CREATE TABLE social_account_shares (
  id INT UNSIGNED NOT NULL AUTO_INCREMENT,
  social_account_id INT UNSIGNED NOT NULL,
  user_account_id INT UNSIGNED NOT NULL,
  can_read TINYINT(1) NOT NULL DEFAULT 1,
  can_post TINYINT(1) NOT NULL DEFAULT 0,
  can_manage TINYINT(1) NOT NULL DEFAULT 0,
  created_at DATETIME(3) DEFAULT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY idx_sa_share (social_account_id, user_account_id),
  CONSTRAINT fk_sas_sa FOREIGN KEY (social_account_id)
    REFERENCES social_accounts(id) ON DELETE CASCADE,
  CONSTRAINT fk_sas_user FOREIGN KEY (user_account_id)
    REFERENCES user_accounts(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb3 COLLATE=utf8mb3_bin;

INSERT IGNORE INTO rbac_permissions (created_at, updated_at, name, description) VALUES
(NOW(3), NOW(3), 'social-accounts.view-all', 'View and manage all social accounts regardless of ownership'),
(NOW(3), NOW(3), 'system.settings', 'View and modify system settings (cvars)'),
(NOW(3), NOW(3), 'system.connectors', 'View connectors and manage OAuth flows'),
(NOW(3), NOW(3), 'system.logs', 'View logs, job status, and run maintenance tasks');

INSERT IGNORE INTO rbac_role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM rbac_roles r JOIN rbac_permissions p ON p.name = 'social-accounts.view-all' WHERE r.name = 'superuser';

INSERT IGNORE INTO rbac_role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM rbac_roles r JOIN rbac_permissions p ON p.name = 'system.settings' WHERE r.name = 'superuser';

INSERT IGNORE INTO rbac_role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM rbac_roles r JOIN rbac_permissions p ON p.name = 'system.connectors' WHERE r.name = 'superuser';

INSERT IGNORE INTO rbac_role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM rbac_roles r JOIN rbac_permissions p ON p.name = 'system.logs' WHERE r.name = 'superuser';
