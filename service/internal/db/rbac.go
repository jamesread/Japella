package db

import (
	"database/sql"
	"fmt"

	"github.com/jamesread/japella/internal/rbac"
)

type RBACPermission struct {
	Model
	Name        string `db:"name"`
	Description string `db:"description"`
}

type RBACRole struct {
	Model
	Name        string `db:"name"`
	Description string `db:"description"`
}

// EffectiveRBAC is the resolved access for a user after loading roles from the database.
type EffectiveRBAC struct {
	IsSuperuser  bool
	Permissions  map[string]bool
	RoleNames    []string
}

func (e *EffectiveRBAC) Has(p string) bool {
	if e == nil {
		return false
	}
	if e.IsSuperuser {
		return true
	}
	return e.Permissions[p]
}

func (db *DB) EnsureRBACBootstrap() error {
	_, err := db.ResilientExec(`
		INSERT IGNORE INTO rbac_user_roles (user_account_id, role_id)
		SELECT u.id, r.id FROM user_accounts u
		CROSS JOIN rbac_roles r
		WHERE u.id = (SELECT MIN(id) FROM user_accounts) AND r.name = ?`,
		rbac.RoleSuperuser)
	if err != nil {
		db.Logger().Errorf("RBAC bootstrap (superuser): %v", err)
		return err
	}

	_, err = db.ResilientExec(`
		INSERT IGNORE INTO rbac_user_roles (user_account_id, role_id)
		SELECT u.id, r.id FROM user_accounts u
		CROSS JOIN rbac_roles r
		WHERE r.name = ?
		AND NOT EXISTS (SELECT 1 FROM rbac_user_roles ur WHERE ur.user_account_id = u.id)`,
		rbac.RoleMember)
	if err != nil {
		db.Logger().Errorf("RBAC bootstrap (member): %v", err)
		return err
	}

	return nil
}

func (db *DB) LoadEffectiveRBAC(userID uint32) (*EffectiveRBAC, error) {
	var superCount int
	err := db.ResilientGet(&superCount, `
		SELECT COUNT(*) FROM rbac_user_roles ur
		INNER JOIN rbac_roles r ON r.id = ur.role_id
		WHERE ur.user_account_id = ? AND r.name = ?`, userID, rbac.RoleSuperuser)
	if err != nil {
		return nil, err
	}

	out := &EffectiveRBAC{
		IsSuperuser: superCount > 0,
		Permissions: make(map[string]bool),
	}

	var roleNames []string
	err = db.ResilientSelect(&roleNames, `
		SELECT r.name FROM rbac_roles r
		INNER JOIN rbac_user_roles ur ON ur.role_id = r.id
		WHERE ur.user_account_id = ?
		ORDER BY r.name`, userID)
	if err != nil {
		return nil, err
	}
	out.RoleNames = roleNames

	if out.IsSuperuser {
		var names []string
		err = db.ResilientSelect(&names, `SELECT name FROM rbac_permissions`)
		if err != nil {
			return nil, err
		}
		for _, n := range names {
			out.Permissions[n] = true
		}
		return out, nil
	}

	var perms []string
	err = db.ResilientSelect(&perms, `
		SELECT DISTINCT p.name FROM rbac_permissions p
		INNER JOIN rbac_role_permissions rp ON rp.permission_id = p.id
		INNER JOIN rbac_user_roles ur ON ur.role_id = rp.role_id
		WHERE ur.user_account_id = ?`, userID)
	if err != nil {
		return nil, err
	}
	for _, p := range perms {
		out.Permissions[p] = true
	}
	return out, nil
}

func (db *DB) SelectRBACPermissions() ([]*RBACPermission, error) {
	ret := make([]*RBACPermission, 0)
	err := db.ResilientSelect(&ret, `SELECT * FROM rbac_permissions ORDER BY name`)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (db *DB) SelectRBACRoles() ([]*RBACRole, error) {
	ret := make([]*RBACRole, 0)
	err := db.ResilientSelect(&ret, `SELECT * FROM rbac_roles ORDER BY name`)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (db *DB) SelectPermissionIDsForRole(roleID uint32) ([]uint32, error) {
	var ids []uint32
	err := db.ResilientSelect(&ids, `SELECT permission_id FROM rbac_role_permissions WHERE role_id = ? ORDER BY permission_id`, roleID)
	return ids, err
}

func (db *DB) GetRBACRoleByID(id uint32) *RBACRole {
	var r RBACRole
	err := db.ResilientGet(&r, `SELECT * FROM rbac_roles WHERE id = ? LIMIT 1`, id)
	if err != nil {
		if err != sql.ErrNoRows {
			db.Logger().Errorf("GetRBACRoleByID: %v", err)
		}
		return nil
	}
	return &r
}

func (db *DB) CreateRBACRole(name, description string) (uint32, error) {
	res, err := db.ResilientExec(
		`INSERT INTO rbac_roles (name, description, created_at, updated_at) VALUES (?, ?, NOW(3), NOW(3))`,
		name, description)
	if err != nil {
		return 0, err
	}
	lid, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	return uint32(lid), nil
}

func (db *DB) UpdateRBACRole(id uint32, name, description string) error {
	role := db.GetRBACRoleByID(id)
	if role == nil {
		return sql.ErrNoRows
	}
	if role.Name == rbac.RoleSuperuser {
		return fmt.Errorf("cannot rename system role %s", rbac.RoleSuperuser)
	}
	if role.Name == rbac.RoleMember && name != rbac.RoleMember {
		return fmt.Errorf("cannot rename system role %s", rbac.RoleMember)
	}
	_, err := db.ResilientExec(`UPDATE rbac_roles SET name = ?, description = ?, updated_at = NOW(3) WHERE id = ?`, name, description, id)
	return err
}

func (db *DB) DeleteRBACRole(id uint32) error {
	role := db.GetRBACRoleByID(id)
	if role == nil {
		return sql.ErrNoRows
	}
	if role.Name == rbac.RoleSuperuser || role.Name == rbac.RoleMember {
		return fmt.Errorf("cannot delete system role %s", role.Name)
	}
	_, err := db.ResilientExec(`DELETE FROM rbac_roles WHERE id = ?`, id)
	return err
}

func (db *DB) SetRBACRolePermissions(roleID uint32, permissionIDs []uint32) error {
	role := db.GetRBACRoleByID(roleID)
	if role == nil {
		return sql.ErrNoRows
	}
	if role.Name == rbac.RoleSuperuser {
		return fmt.Errorf("cannot change permissions for role %s", rbac.RoleSuperuser)
	}

	tx, err := db.connx.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err = tx.Exec(`DELETE FROM rbac_role_permissions WHERE role_id = ?`, roleID); err != nil {
		return err
	}
	for _, pid := range permissionIDs {
		if _, err = tx.Exec(`INSERT INTO rbac_role_permissions (role_id, permission_id) VALUES (?, ?)`, roleID, pid); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (db *DB) SelectUserRBACRoleIDs(userID uint32) ([]uint32, error) {
	var ids []uint32
	err := db.ResilientSelect(&ids, `SELECT role_id FROM rbac_user_roles WHERE user_account_id = ? ORDER BY role_id`, userID)
	return ids, err
}

func (db *DB) SetUserRBACRoles(targetUserID uint32, roleIDs []uint32) error {
	if targetUserID == 0 {
		return fmt.Errorf("invalid user id")
	}
	if db.connx == nil {
		db.ReconnectDatabaseAndSetErrorMessage()
		return fmt.Errorf("database connection is not established")
	}

	tx, err := db.connx.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err = tx.Exec(`DELETE FROM rbac_user_roles WHERE user_account_id = ?`, targetUserID); err != nil {
		return err
	}
	for _, rid := range roleIDs {
		if _, err = tx.Exec(`INSERT INTO rbac_user_roles (user_account_id, role_id) VALUES (?, ?)`, targetUserID, rid); err != nil {
			return err
		}
	}

	// Prevent removing last superuser in the system
	var superCount int
	if err = tx.Get(&superCount, `
		SELECT COUNT(*) FROM rbac_user_roles ur
		INNER JOIN rbac_roles r ON r.id = ur.role_id
		WHERE r.name = ?`, rbac.RoleSuperuser); err != nil {
		return err
	}
	if superCount == 0 {
		return fmt.Errorf("refusing to leave the system without a superuser")
	}

	return tx.Commit()
}

func (db *DB) AssignRBACRoleByName(userID uint32, roleName string) error {
	var roleID uint32
	err := db.ResilientGet(&roleID, `SELECT id FROM rbac_roles WHERE name = ? LIMIT 1`, roleName)
	if err != nil {
		return err
	}
	_, err = db.ResilientExec(
		`INSERT IGNORE INTO rbac_user_roles (user_account_id, role_id) VALUES (?, ?)`,
		userID, roleID)
	return err
}
