package iam

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jamesread/armature-iam/rbac"
	"github.com/jamesread/armature-iam/store"
)

func (s *DBStore) LoadEffectiveRBAC(ctx context.Context, userID int) (*rbac.EffectiveRBAC, error) {
	var superCount int
	err := s.DB.ResilientGet(&superCount, `
		SELECT COUNT(*) FROM user_group_memberships ugm
		INNER JOIN rbac_group_roles gr ON gr.user_group_id = ugm.user_group_id
		INNER JOIN rbac_roles r ON r.id = gr.role_id
		WHERE ugm.user_account_id = ? AND r.name = ?`, userID, rbac.RoleSuperuser)
	if err != nil {
		return nil, err
	}

	out := &rbac.EffectiveRBAC{
		IsSuperuser: superCount > 0,
		Permissions: map[string]bool{},
	}

	var roleNames []string
	err = s.DB.ResilientSelect(&roleNames, `
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
		err = s.DB.ResilientSelect(&names, `SELECT name FROM rbac_permissions ORDER BY name`)
		if err != nil {
			return nil, err
		}
		for _, n := range names {
			out.Permissions[n] = true
		}
		return out, nil
	}

	var perms []string
	err = s.DB.ResilientSelect(&perms, `
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

func (s *DBStore) EnsureRBACBootstrap(ctx context.Context) error {
	everyoneID, err := s.ensureSystemGroup(ctx, rbac.GroupEveryone)
	if err != nil {
		return err
	}
	administratorsID, err := s.ensureSystemGroup(ctx, rbac.GroupAdministrators)
	if err != nil {
		return err
	}

	var memberRoleID, superuserRoleID int
	if err := s.DB.ResilientGet(&memberRoleID, `SELECT id FROM rbac_roles WHERE name = ? LIMIT 1`, rbac.RoleMember); err != nil {
		return err
	}
	if err := s.DB.ResilientGet(&superuserRoleID, `SELECT id FROM rbac_roles WHERE name = ? LIMIT 1`, rbac.RoleSuperuser); err != nil {
		return err
	}

	if err := s.ensureGroupHasRole(ctx, everyoneID, memberRoleID); err != nil {
		return err
	}
	if err := s.ensureGroupHasRole(ctx, administratorsID, superuserRoleID); err != nil {
		return err
	}

	count, err := s.CountUsersWithSuperuserViaGroups(ctx)
	if err != nil {
		return err
	}
	if count == 0 {
		_, err = s.DB.ResilientExec(`
			INSERT IGNORE INTO user_group_memberships (user_account_id, user_group_id, created_at, updated_at)
			SELECT u.id, ?, NOW(3), NOW(3) FROM user_accounts u
			ORDER BY u.id LIMIT 1`, administratorsID)
		if err != nil {
			return err
		}
	}

	_, err = s.DB.ResilientExec(`
		INSERT IGNORE INTO user_group_memberships (user_account_id, user_group_id, created_at, updated_at)
		SELECT u.id, ?, NOW(3), NOW(3) FROM user_accounts u
		WHERE NOT EXISTS (
			SELECT 1 FROM user_group_memberships ugm WHERE ugm.user_account_id = u.id
		)`, everyoneID)
	return err
}

func (s *DBStore) ensureSystemGroup(ctx context.Context, name string) (int, error) {
	g := s.DB.GetUserGroupByName(name)
	if g != nil {
		return int(g.ID), nil
	}
	return s.CreateUserGroup(ctx, name)
}

func (s *DBStore) ensureGroupHasRole(ctx context.Context, groupID, roleID int) error {
	_, err := s.DB.ResilientExec(
		`INSERT IGNORE INTO rbac_group_roles (user_group_id, role_id) VALUES (?, ?)`,
		groupID, roleID)
	return err
}

func (s *DBStore) ensureSuperuserCoverage(ctx context.Context) error {
	count, err := s.CountUsersWithSuperuserViaGroups(ctx)
	if err != nil {
		return err
	}
	if count == 0 {
		return store.ErrNoSuperuser
	}
	return nil
}

func (s *DBStore) ListRBACPermissions(ctx context.Context) ([]store.RBACPermissionRow, error) {
	rows, err := s.DB.SelectRBACPermissions()
	if err != nil {
		return nil, err
	}
	out := make([]store.RBACPermissionRow, 0, len(rows))
	for _, p := range rows {
		out = append(out, store.RBACPermissionRow{
			ID:          int(p.ID),
			Name:        p.Name,
			Description: p.Description,
		})
	}
	return out, nil
}

func (s *DBStore) ListRBACRoles(ctx context.Context) ([]store.RBACRoleRow, error) {
	roles, err := s.DB.SelectRBACRoles()
	if err != nil {
		return nil, err
	}
	usage, err := s.DB.SelectRBACRoleUsageStats()
	if err != nil {
		return nil, err
	}
	out := make([]store.RBACRoleRow, 0, len(roles))
	for _, r := range roles {
		permIDs, err := s.ListRolePermissionIDs(ctx, int(r.ID))
		if err != nil {
			return nil, err
		}
		u := usage[r.ID]
		out = append(out, store.RBACRoleRow{
			ID:            int(r.ID),
			Name:          r.Name,
			Description:   r.Description,
			PermissionIDs: permIDs,
			GroupCount:    int(u.GroupCount),
			UserCount:     int(u.UserCount),
		})
	}
	return out, nil
}

func (s *DBStore) GetRBACRole(ctx context.Context, id int) (*store.RBACRoleRow, error) {
	role := s.DB.GetRBACRoleByID(uint32(id))
	if role == nil {
		return nil, nil
	}
	permIDs, err := s.ListRolePermissionIDs(ctx, id)
	if err != nil {
		return nil, err
	}
	usage, err := s.DB.SelectRBACRoleUsageStats()
	if err != nil {
		return nil, err
	}
	u := usage[role.ID]
	return &store.RBACRoleRow{
		ID:            int(role.ID),
		Name:          role.Name,
		Description:   role.Description,
		PermissionIDs: permIDs,
		GroupCount:    int(u.GroupCount),
		UserCount:     int(u.UserCount),
	}, nil
}

func (s *DBStore) CreateRBACRole(ctx context.Context, name, description string, permissionIDs []int) (int, error) {
	id, err := s.DB.CreateRBACRole(name, description)
	if err != nil {
		return 0, err
	}
	if err := s.DB.SetRBACRolePermissions(id, intSliceToUint32(permissionIDs)); err != nil {
		_, _ = s.DB.ResilientExec(`DELETE FROM rbac_roles WHERE id = ?`, id)
		return 0, err
	}
	return int(id), nil
}

func (s *DBStore) UpdateRBACRole(ctx context.Context, id int, name, description string, permissionIDs []int) error {
	cur := s.DB.GetRBACRoleByID(uint32(id))
	if cur == nil {
		return sql.ErrNoRows
	}
	if rbac.IsSystemRole(cur.Name) && name != cur.Name {
		return fmt.Errorf("%w %q", store.ErrRenameSystemRole, cur.Name)
	}
	if err := s.DB.UpdateRBACRole(uint32(id), name, description); err != nil {
		return err
	}
	if cur.Name == rbac.RoleSuperuser {
		return nil
	}
	return s.DB.SetRBACRolePermissions(uint32(id), intSliceToUint32(permissionIDs))
}

func (s *DBStore) DeleteRBACRole(ctx context.Context, id int) error {
	role := s.DB.GetRBACRoleByID(uint32(id))
	if role == nil {
		return sql.ErrNoRows
	}
	if rbac.IsSystemRole(role.Name) {
		return fmt.Errorf("%w %q", store.ErrSystemRole, role.Name)
	}
	return s.DB.DeleteRBACRole(uint32(id))
}

func (s *DBStore) SetRBACRolePermissions(ctx context.Context, roleID int, permissionIDs []int) error {
	role := s.DB.GetRBACRoleByID(uint32(roleID))
	if role == nil {
		return sql.ErrNoRows
	}
	if role.Name == rbac.RoleSuperuser {
		return fmt.Errorf("%w %q", store.ErrSystemRole, role.Name)
	}
	return s.DB.SetRBACRolePermissions(uint32(roleID), intSliceToUint32(permissionIDs))
}

func (s *DBStore) ListRolePermissionIDs(ctx context.Context, roleID int) ([]int, error) {
	ids, err := s.DB.SelectPermissionIDsForRole(uint32(roleID))
	if err != nil {
		return nil, err
	}
	return uint32SliceToInt(ids), nil
}

func (s *DBStore) ListPermissionRoleNames(ctx context.Context, permissionID int) ([]string, error) {
	var names []string
	err := s.DB.ResilientSelect(&names, `
		SELECT r.name FROM rbac_roles r
		INNER JOIN rbac_role_permissions rp ON rp.role_id = r.id
		WHERE rp.permission_id = ?
		ORDER BY r.name`, permissionID)
	return names, err
}

func (s *DBStore) GetUserRbacRoleNames(ctx context.Context, userID int) ([]string, error) {
	var names []string
	err := s.DB.ResilientSelect(&names, `
		SELECT DISTINCT r.name FROM rbac_roles r
		INNER JOIN rbac_group_roles gr ON gr.role_id = r.id
		INNER JOIN user_group_memberships ugm ON ugm.user_group_id = gr.user_group_id
		WHERE ugm.user_account_id = ?
		ORDER BY r.name`, userID)
	return names, err
}

func (s *DBStore) GetUserGroupRbacRoleIDs(ctx context.Context, groupID int) ([]int, error) {
	ids, err := s.DB.SelectRBACRoleIDsForGroup(uint32(groupID))
	if err != nil {
		return nil, err
	}
	return uint32SliceToInt(ids), nil
}

func (s *DBStore) SetUserGroupRbacRoles(ctx context.Context, groupID int, roleIDs []int) error {
	if err := s.DB.SetRBACGroupRoles(uint32(groupID), intSliceToUint32(roleIDs)); err != nil {
		return err
	}
	return s.ensureSuperuserCoverage(ctx)
}

func (s *DBStore) ListRbacRoleGroupNames(ctx context.Context, roleID int) ([]string, error) {
	var names []string
	err := s.DB.ResilientSelect(&names, `
		SELECT g.name FROM user_groups g
		INNER JOIN rbac_group_roles gr ON gr.user_group_id = g.id
		WHERE gr.role_id = ?
		ORDER BY g.name`, roleID)
	return names, err
}

func (s *DBStore) ListRbacRoleUsernames(ctx context.Context, roleID int) ([]string, error) {
	var names []string
	err := s.DB.ResilientSelect(&names, `
		SELECT DISTINCT u.username FROM user_accounts u
		INNER JOIN user_group_memberships ugm ON ugm.user_account_id = u.id
		INNER JOIN rbac_group_roles gr ON gr.user_group_id = ugm.user_group_id
		WHERE gr.role_id = ?
		ORDER BY u.username`, roleID)
	return names, err
}

func (s *DBStore) GetMyPermissionsAudit(ctx context.Context, userID int) ([]string, []string, bool, []store.MyPermissionAuditRow, error) {
	audit, err := s.DB.BuildUserPermissionsAudit(uint32(userID))
	if err != nil {
		return nil, nil, false, nil, err
	}
	rows := make([]store.MyPermissionAuditRow, 0, len(audit.Rows))
	for _, row := range audit.Rows {
		rows = append(rows, store.MyPermissionAuditRow{
			Permission:     row.Permission,
			Granted:        row.Granted,
			GrantingGroups: row.GrantingGroups,
		})
	}
	return audit.GroupNames, audit.RoleNames, audit.IsSuperuser, rows, nil
}
