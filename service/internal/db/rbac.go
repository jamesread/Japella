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
	IsSuperuser bool
	Permissions map[string]bool
	RoleNames   []string
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

func (db *DB) ensureSystemGroup(name string) (uint32, error) {
	g := db.GetUserGroupByName(name)
	if g != nil {
		return g.ID, nil
	}
	return db.CreateUserGroup(name)
}

func (db *DB) ensureGroupHasRole(groupID, roleID uint32) error {
	_, err := db.ResilientExec(
		`INSERT IGNORE INTO rbac_group_roles (user_group_id, role_id) VALUES (?, ?)`,
		groupID, roleID)
	return err
}

func (db *DB) ensureUserInGroup(userID, groupID uint32) error {
	return db.AddUserGroupMember(userID, groupID)
}

func (db *DB) CountUsersWithSuperuserViaGroups() (int, error) {
	var count int
	err := db.ResilientGet(&count, `
		SELECT COUNT(DISTINCT ugm.user_account_id)
		FROM user_group_memberships ugm
		INNER JOIN rbac_group_roles gr ON gr.user_group_id = ugm.user_group_id
		INNER JOIN rbac_roles r ON r.id = gr.role_id
		WHERE r.name = ?`, rbac.RoleSuperuser)
	return count, err
}

func (db *DB) ensureSuperuserCoverage() error {
	count, err := db.CountUsersWithSuperuserViaGroups()
	if err != nil {
		return err
	}
	if count == 0 {
		return fmt.Errorf("refusing to leave the system without a superuser")
	}
	return nil
}

func (db *DB) EnsureRBACBootstrap() error {
	everyoneID, err := db.ensureSystemGroup(rbac.GroupEveryone)
	if err != nil {
		db.Logger().Errorf("RBAC bootstrap (Everyone group): %v", err)
		return err
	}
	administratorsID, err := db.ensureSystemGroup(rbac.GroupAdministrators)
	if err != nil {
		db.Logger().Errorf("RBAC bootstrap (Administrators group): %v", err)
		return err
	}

	var memberRoleID, superuserRoleID uint32
	if err := db.ResilientGet(&memberRoleID, `SELECT id FROM rbac_roles WHERE name = ? LIMIT 1`, rbac.RoleMember); err != nil {
		return err
	}
	if err := db.ResilientGet(&superuserRoleID, `SELECT id FROM rbac_roles WHERE name = ? LIMIT 1`, rbac.RoleSuperuser); err != nil {
		return err
	}

	if err := db.ensureGroupHasRole(everyoneID, memberRoleID); err != nil {
		db.Logger().Errorf("RBAC bootstrap (Everyone member role): %v", err)
		return err
	}
	if err := db.ensureGroupHasRole(administratorsID, superuserRoleID); err != nil {
		db.Logger().Errorf("RBAC bootstrap (Administrators superuser role): %v", err)
		return err
	}

	superCount, err := db.CountUsersWithSuperuserViaGroups()
	if err != nil {
		return err
	}
	if superCount == 0 {
		var firstUserID uint32
		if err := db.ResilientGet(&firstUserID, `SELECT MIN(id) FROM user_accounts`); err == nil && firstUserID > 0 {
			if err := db.ensureUserInGroup(firstUserID, administratorsID); err != nil {
				db.Logger().Errorf("RBAC bootstrap (first user Administrators): %v", err)
				return err
			}
		}
	}

	_, err = db.ResilientExec(`
		INSERT IGNORE INTO user_group_memberships (user_account_id, user_group_id, created_at, updated_at)
		SELECT u.id, ?, NOW(3), NOW(3) FROM user_accounts u
		WHERE NOT EXISTS (
			SELECT 1 FROM user_group_memberships ugm WHERE ugm.user_account_id = u.id
		)`, everyoneID)
	if err != nil {
		db.Logger().Errorf("RBAC bootstrap (Everyone membership): %v", err)
		return err
	}

	return nil
}

func (db *DB) EnsureUserInEveryoneGroup(userID uint32) error {
	if userID == 0 {
		return fmt.Errorf("invalid user id")
	}
	everyoneID, err := db.ensureSystemGroup(rbac.GroupEveryone)
	if err != nil {
		return err
	}
	var memberRoleID uint32
	if err := db.ResilientGet(&memberRoleID, `SELECT id FROM rbac_roles WHERE name = ? LIMIT 1`, rbac.RoleMember); err != nil {
		return err
	}
	if err := db.ensureGroupHasRole(everyoneID, memberRoleID); err != nil {
		return err
	}
	return db.ensureUserInGroup(userID, everyoneID)
}

func (db *DB) LoadEffectiveRBAC(userID uint32) (*EffectiveRBAC, error) {
	var superCount int
	err := db.ResilientGet(&superCount, `
		SELECT COUNT(*) FROM user_group_memberships ugm
		INNER JOIN rbac_group_roles gr ON gr.user_group_id = ugm.user_group_id
		INNER JOIN rbac_roles r ON r.id = gr.role_id
		WHERE ugm.user_account_id = ? AND r.name = ?`, userID, rbac.RoleSuperuser)
	if err != nil {
		return nil, err
	}

	out := &EffectiveRBAC{
		IsSuperuser: superCount > 0,
		Permissions: make(map[string]bool),
	}

	var roleNames []string
	err = db.ResilientSelect(&roleNames, `
		SELECT DISTINCT r.name FROM rbac_roles r
		INNER JOIN rbac_group_roles gr ON gr.role_id = r.id
		INNER JOIN user_group_memberships ugm ON ugm.user_group_id = gr.user_group_id
		WHERE ugm.user_account_id = ?
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
		INNER JOIN rbac_group_roles gr ON gr.role_id = rp.role_id
		INNER JOIN user_group_memberships ugm ON ugm.user_group_id = gr.user_group_id
		WHERE ugm.user_account_id = ?`, userID)
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

type RBACRoleUsage struct {
	RoleID     uint32 `db:"role_id"`
	GroupCount uint32 `db:"group_count"`
	UserCount  uint32 `db:"user_count"`
}

func (db *DB) SelectRBACRoleUsageStats() (map[uint32]RBACRoleUsage, error) {
	rows := make([]RBACRoleUsage, 0)
	err := db.ResilientSelect(&rows, `
		SELECT gr.role_id,
		       COUNT(DISTINCT gr.user_group_id) AS group_count,
		       COUNT(DISTINCT ugm.user_account_id) AS user_count
		FROM rbac_group_roles gr
		LEFT JOIN user_group_memberships ugm ON ugm.user_group_id = gr.user_group_id
		GROUP BY gr.role_id`)
	if err != nil {
		return nil, err
	}
	out := make(map[uint32]RBACRoleUsage, len(rows))
	for _, row := range rows {
		out[row.RoleID] = row
	}
	return out, nil
}

func (db *DB) SelectUserRBACRoleIDs(userID uint32) ([]uint32, error) {
	var ids []uint32
	err := db.ResilientSelect(&ids, `
		SELECT DISTINCT gr.role_id
		FROM user_group_memberships ugm
		INNER JOIN rbac_group_roles gr ON gr.user_group_id = ugm.user_group_id
		WHERE ugm.user_account_id = ?
		ORDER BY gr.role_id`, userID)
	return ids, err
}

func (db *DB) SelectRBACRoleIDsForGroup(groupID uint32) ([]uint32, error) {
	var ids []uint32
	err := db.ResilientSelect(&ids, `SELECT role_id FROM rbac_group_roles WHERE user_group_id = ? ORDER BY role_id`, groupID)
	return ids, err
}

func (db *DB) SelectGroupIDsForRBACRole(roleID uint32) ([]uint32, error) {
	var ids []uint32
	err := db.ResilientSelect(&ids, `SELECT user_group_id FROM rbac_group_roles WHERE role_id = ? ORDER BY user_group_id`, roleID)
	return ids, err
}

func (db *DB) SelectUserIDsWithRoleViaGroups(roleID uint32) ([]uint32, error) {
	var ids []uint32
	err := db.ResilientSelect(&ids, `
		SELECT DISTINCT ugm.user_account_id
		FROM user_group_memberships ugm
		INNER JOIN rbac_group_roles gr ON gr.user_group_id = ugm.user_group_id
		WHERE gr.role_id = ?
		ORDER BY ugm.user_account_id`, roleID)
	return ids, err
}

func (db *DB) SetRBACGroupRoles(groupID uint32, roleIDs []uint32) error {
	if groupID == 0 {
		return fmt.Errorf("invalid group id")
	}
	if db.GetUserGroupByID(groupID) == nil {
		return sql.ErrNoRows
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

	if _, err = tx.Exec(`DELETE FROM rbac_group_roles WHERE user_group_id = ?`, groupID); err != nil {
		return err
	}
	for _, rid := range roleIDs {
		if _, err = tx.Exec(`INSERT INTO rbac_group_roles (user_group_id, role_id) VALUES (?, ?)`, groupID, rid); err != nil {
			return err
		}
	}

	if err = tx.Commit(); err != nil {
		return err
	}
	return db.ensureSuperuserCoverage()
}

type UserPermissionsAuditRow struct {
	Permission     string
	Granted        bool
	GrantingGroups []string
}

type UserPermissionsAudit struct {
	GroupNames  []string
	RoleNames   []string
	IsSuperuser bool
	Rows        []UserPermissionsAuditRow
}

func (db *DB) BuildUserPermissionsAudit(userID uint32) (*UserPermissionsAudit, error) {
	effective, err := db.LoadEffectiveRBAC(userID)
	if err != nil {
		return nil, err
	}

	var groupNames []string
	err = db.ResilientSelect(&groupNames, `
		SELECT ug.name FROM user_groups ug
		INNER JOIN user_group_memberships ugm ON ugm.user_group_id = ug.id
		WHERE ugm.user_account_id = ?
		ORDER BY ug.name`, userID)
	if err != nil {
		return nil, err
	}

	allPerms, err := db.SelectRBACPermissions()
	if err != nil {
		return nil, err
	}

	grantingByPerm := make(map[string][]string)
	if !effective.IsSuperuser {
		type permGroupRow struct {
			Permission string `db:"permission"`
			GroupName  string `db:"group_name"`
		}
		var grantRows []permGroupRow
		err = db.ResilientSelect(&grantRows, `
			SELECT DISTINCT p.name AS permission, ug.name AS group_name
			FROM user_group_memberships ugm
			INNER JOIN user_groups ug ON ug.id = ugm.user_group_id
			INNER JOIN rbac_group_roles gr ON gr.user_group_id = ug.id
			INNER JOIN rbac_role_permissions rp ON rp.role_id = gr.role_id
			INNER JOIN rbac_permissions p ON p.id = rp.permission_id
			WHERE ugm.user_account_id = ?
			ORDER BY p.name, ug.name`, userID)
		if err != nil {
			return nil, err
		}
		for _, row := range grantRows {
			grantingByPerm[row.Permission] = append(grantingByPerm[row.Permission], row.GroupName)
		}
	}

	rows := make([]UserPermissionsAuditRow, 0, len(allPerms))
	for _, p := range allPerms {
		granting := grantingByPerm[p.Name]
		granted := effective.IsSuperuser || len(granting) > 0
		rows = append(rows, UserPermissionsAuditRow{
			Permission:     p.Name,
			Granted:        granted,
			GrantingGroups: granting,
		})
	}

	return &UserPermissionsAudit{
		GroupNames:  groupNames,
		RoleNames:   effective.RoleNames,
		IsSuperuser: effective.IsSuperuser,
		Rows:        rows,
	}, nil
}
