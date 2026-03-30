ALTER TABLE sessions
  ADD COLUMN impersonator_user_id INT UNSIGNED DEFAULT NULL,
  ADD CONSTRAINT fk_sessions_impersonator FOREIGN KEY (impersonator_user_id)
    REFERENCES user_accounts(id) ON DELETE SET NULL;

INSERT IGNORE INTO rbac_permissions (created_at, updated_at, name, description) VALUES
(NOW(3), NOW(3), 'system.impersonate', 'Impersonate other users to debug permission issues');

INSERT IGNORE INTO rbac_role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM rbac_roles r JOIN rbac_permissions p ON p.name = 'system.impersonate' WHERE r.name = 'superuser';
