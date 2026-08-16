package api

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	controlv1 "github.com/jamesread/japella/gen/japella/controlapi/v1"
	"github.com/jamesread/japella/internal/rbac"
	log "github.com/sirupsen/logrus"
)

func (s *ControlApi) ListRbacPermissions(ctx context.Context, req *connect.Request[controlv1.ListRbacPermissionsRequest]) (*connect.Response[controlv1.ListRbacPermissionsResponse], error) {
	perms, err := s.DB.SelectRBACPermissions()
	if err != nil {
		log.Errorf("ListRbacPermissions: %v", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to list permissions"))
	}
	out := make([]*controlv1.RbacPermission, 0, len(perms))
	for _, p := range perms {
		out = append(out, &controlv1.RbacPermission{
			Id:          p.ID,
			Name:        p.Name,
			Description: p.Description,
		})
	}
	return connect.NewResponse(&controlv1.ListRbacPermissionsResponse{Permissions: out}), nil
}

func (s *ControlApi) ListRbacRoles(ctx context.Context, req *connect.Request[controlv1.ListRbacRolesRequest]) (*connect.Response[controlv1.ListRbacRolesResponse], error) {
	roles, err := s.DB.SelectRBACRoles()
	if err != nil {
		log.Errorf("ListRbacRoles: %v", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to list roles"))
	}
	usage, err := s.DB.SelectRBACRoleUsageStats()
	if err != nil {
		log.Errorf("ListRbacRoles usage: %v", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to load role usage"))
	}
	out := make([]*controlv1.RbacRole, 0, len(roles))
	for _, r := range roles {
		ids, err := s.DB.SelectPermissionIDsForRole(r.ID)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to load role permissions"))
		}
		u := usage[r.ID]
		out = append(out, &controlv1.RbacRole{
			Id:            r.ID,
			Name:          r.Name,
			Description:   r.Description,
			PermissionIds: ids,
			GroupCount:    u.GroupCount,
			UserCount:     u.UserCount,
		})
	}
	return connect.NewResponse(&controlv1.ListRbacRolesResponse{Roles: out}), nil
}

func (s *ControlApi) CreateRbacRole(ctx context.Context, req *connect.Request[controlv1.CreateRbacRoleRequest]) (*connect.Response[controlv1.CreateRbacRoleResponse], error) {
	name := strings.TrimSpace(req.Msg.Name)
	if name == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("name is required"))
	}
	if name == rbac.RoleSuperuser || name == rbac.RoleMember {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("reserved role name"))
	}

	id, err := s.DB.CreateRBACRole(name, strings.TrimSpace(req.Msg.Description))
	if err != nil {
		log.Errorf("CreateRbacRole: %v", err)
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("could not create role (duplicate name or invalid data)"))
	}
	if err := s.DB.SetRBACRolePermissions(id, req.Msg.PermissionIds); err != nil {
		log.Errorf("CreateRbacRole permissions: %v", err)
		_, _ = s.DB.ResilientExec(`DELETE FROM rbac_roles WHERE id = ?`, id)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to set role permissions"))
	}
	return connect.NewResponse(&controlv1.CreateRbacRoleResponse{
		StandardResponse: &controlv1.StandardResponse{Success: true, Message: "Role created"},
		RoleId:           id,
	}), nil
}

func (s *ControlApi) UpdateRbacRole(ctx context.Context, req *connect.Request[controlv1.UpdateRbacRoleRequest]) (*connect.Response[controlv1.UpdateRbacRoleResponse], error) {
	if req.Msg.RoleId == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("role_id is required"))
	}
	name := strings.TrimSpace(req.Msg.Name)
	if name == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("name is required"))
	}
	role := s.DB.GetRBACRoleByID(req.Msg.RoleId)
	if role == nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("role not found"))
	}
	if role.Name == rbac.RoleSuperuser {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("cannot modify system role %s", rbac.RoleSuperuser))
	}
	if name == rbac.RoleSuperuser || (role.Name != rbac.RoleMember && name == rbac.RoleMember) {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("reserved role name"))
	}

	if err := s.DB.UpdateRBACRole(req.Msg.RoleId, name, strings.TrimSpace(req.Msg.Description)); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("role not found"))
		}
		log.Errorf("UpdateRbacRole: %v", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to update role"))
	}
	if err := s.DB.SetRBACRolePermissions(req.Msg.RoleId, req.Msg.PermissionIds); err != nil {
		log.Errorf("UpdateRbacRole permissions: %v", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to set role permissions"))
	}
	return connect.NewResponse(&controlv1.UpdateRbacRoleResponse{
		StandardResponse: &controlv1.StandardResponse{Success: true, Message: "Role updated"},
	}), nil
}

func (s *ControlApi) DeleteRbacRole(ctx context.Context, req *connect.Request[controlv1.DeleteRbacRoleRequest]) (*connect.Response[controlv1.DeleteRbacRoleResponse], error) {
	if req.Msg.RoleId == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("role_id is required"))
	}
	if err := s.DB.DeleteRBACRole(req.Msg.RoleId); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("role not found"))
		}
		log.Errorf("DeleteRbacRole: %v", err)
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&controlv1.DeleteRbacRoleResponse{
		StandardResponse: &controlv1.StandardResponse{Success: true, Message: "Role deleted"},
	}), nil
}

func (s *ControlApi) GetUserRbacRoles(ctx context.Context, req *connect.Request[controlv1.GetUserRbacRolesRequest]) (*connect.Response[controlv1.GetUserRbacRolesResponse], error) {
	if req.Msg.UserId == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("user_id is required"))
	}
	if s.DB.GetUserByID(req.Msg.UserId) == nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("user not found"))
	}
	ids, err := s.DB.SelectUserRBACRoleIDs(req.Msg.UserId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to load user roles"))
	}
	return connect.NewResponse(&controlv1.GetUserRbacRolesResponse{RoleIds: ids}), nil
}

func (s *ControlApi) GetMyPermissionsAudit(ctx context.Context, req *connect.Request[controlv1.GetMyPermissionsAuditRequest]) (*connect.Response[controlv1.GetMyPermissionsAuditResponse], error) {
	authenticatedUser := s.getAuthenticatedUser(ctx)
	if authenticatedUser == nil || authenticatedUser.User == nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}

	audit, err := s.DB.BuildUserPermissionsAudit(authenticatedUser.User.ID)
	if err != nil {
		log.Errorf("GetMyPermissionsAudit: %v", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to load permissions audit"))
	}

	rows := make([]*controlv1.MyPermissionAuditRow, 0, len(audit.Rows))
	for _, row := range audit.Rows {
		rows = append(rows, &controlv1.MyPermissionAuditRow{
			Permission:     row.Permission,
			Granted:        row.Granted,
			GrantingGroups: row.GrantingGroups,
		})
	}

	return connect.NewResponse(&controlv1.GetMyPermissionsAuditResponse{
		GroupNames:  audit.GroupNames,
		RoleNames:   audit.RoleNames,
		IsSuperuser: audit.IsSuperuser,
		Permissions: rows,
	}), nil
}

func (s *ControlApi) GetUserGroupRbacRoles(ctx context.Context, req *connect.Request[controlv1.GetUserGroupRbacRolesRequest]) (*connect.Response[controlv1.GetUserGroupRbacRolesResponse], error) {
	if req.Msg.GroupId == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("group_id is required"))
	}
	if s.DB.GetUserGroupByID(req.Msg.GroupId) == nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("group not found"))
	}
	ids, err := s.DB.SelectRBACRoleIDsForGroup(req.Msg.GroupId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to load group roles"))
	}
	return connect.NewResponse(&controlv1.GetUserGroupRbacRolesResponse{RoleIds: ids}), nil
}

func (s *ControlApi) SetUserGroupRbacRoles(ctx context.Context, req *connect.Request[controlv1.SetUserGroupRbacRolesRequest]) (*connect.Response[controlv1.SetUserGroupRbacRolesResponse], error) {
	if req.Msg.GroupId == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("group_id is required"))
	}
	if s.DB.GetUserGroupByID(req.Msg.GroupId) == nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("group not found"))
	}
	if err := s.DB.SetRBACGroupRoles(req.Msg.GroupId, req.Msg.RoleIds); err != nil {
		log.Errorf("SetUserGroupRbacRoles: %v", err)
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&controlv1.SetUserGroupRbacRolesResponse{
		StandardResponse: &controlv1.StandardResponse{Success: true, Message: "Group roles updated"},
	}), nil
}

func (s *ControlApi) GetRbacRoleUsers(ctx context.Context, req *connect.Request[controlv1.GetRbacRoleUsersRequest]) (*connect.Response[controlv1.GetRbacRoleUsersResponse], error) {
	if req.Msg.RoleId == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("role_id is required"))
	}
	if s.DB.GetRBACRoleByID(req.Msg.RoleId) == nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("role not found"))
	}
	ids, err := s.DB.SelectUserIDsWithRoleViaGroups(req.Msg.RoleId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to load role users"))
	}
	return connect.NewResponse(&controlv1.GetRbacRoleUsersResponse{UserIds: ids}), nil
}

func (s *ControlApi) GetRbacRoleGroups(ctx context.Context, req *connect.Request[controlv1.GetRbacRoleGroupsRequest]) (*connect.Response[controlv1.GetRbacRoleGroupsResponse], error) {
	if req.Msg.RoleId == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("role_id is required"))
	}
	if s.DB.GetRBACRoleByID(req.Msg.RoleId) == nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("role not found"))
	}
	ids, err := s.DB.SelectGroupIDsForRBACRole(req.Msg.RoleId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to load role groups"))
	}
	return connect.NewResponse(&controlv1.GetRbacRoleGroupsResponse{GroupIds: ids}), nil
}
