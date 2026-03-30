package authentication

import (
	"github.com/jamesread/japella/internal/rbac"
	controlv1 "github.com/jamesread/japella/gen/japella/controlapi/v1/controlv1connect"
)

// RequiredPermission returns the permission needed for an authenticated Connect procedure.
// Allow-listed unauthenticated procedures are handled before this runs.
// Default is app.access for normal application use.
func RequiredPermission(procedureName string) string {
	switch procedureName {
	// GetUsers lists id + username for all accounts (pickers, sharing, groups). Any user with
	// app.access may call it; sensitive per-user operations still use GetUser and users.view.
	case controlv1.JapellaControlApiServiceGetUserProcedure:
		return rbac.PermissionUsersView
	case controlv1.JapellaControlApiServiceCreateUserProcedure:
		return rbac.PermissionUsersCreate
	case controlv1.JapellaControlApiServiceDeleteUserProcedure:
		return rbac.PermissionUsersDelete
	case controlv1.JapellaControlApiServiceResetUserPasswordProcedure:
		return rbac.PermissionUsersResetPassword

	case controlv1.JapellaControlApiServiceListRbacPermissionsProcedure,
		controlv1.JapellaControlApiServiceListRbacRolesProcedure,
		controlv1.JapellaControlApiServiceGetUserRbacRolesProcedure:
		return rbac.PermissionRbacView

	case controlv1.JapellaControlApiServiceCreateRbacRoleProcedure,
		controlv1.JapellaControlApiServiceUpdateRbacRoleProcedure,
		controlv1.JapellaControlApiServiceDeleteRbacRoleProcedure,
		controlv1.JapellaControlApiServiceSetUserRbacRolesProcedure:
		return rbac.PermissionRbacManage

	case controlv1.JapellaControlApiServiceGetCvarsProcedure,
		controlv1.JapellaControlApiServiceSetCvarProcedure:
		return rbac.PermissionSystemSettings

	case controlv1.JapellaControlApiServiceGetConnectorsProcedure,
		controlv1.JapellaControlApiServiceRefreshConnectorsProcedure,
		controlv1.JapellaControlApiServiceStartOAuthProcedure,
		controlv1.JapellaControlApiServiceRegisterConnectorProcedure:
		return rbac.PermissionSystemConnectors

	case controlv1.JapellaControlApiServiceGetLogsProcedure,
		controlv1.JapellaControlApiServiceGetJobsStatusProcedure,
		controlv1.JapellaControlApiServiceCleanupFeedPostsProcedure:
		return rbac.PermissionSystemLogs

	case controlv1.JapellaControlApiServiceImpersonateUserProcedure:
		return rbac.PermissionSystemImpersonate

	case controlv1.JapellaControlApiServiceStopImpersonationProcedure:
		return ""

	case controlv1.JapellaControlApiServiceGetUserGroupMembersProcedure:
		return rbac.PermissionUserGroupsView

	case controlv1.JapellaControlApiServiceCreateUserGroupProcedure,
		controlv1.JapellaControlApiServiceDeleteUserGroupProcedure,
		controlv1.JapellaControlApiServiceSetUserGroupMembersProcedure:
		return rbac.PermissionUserGroupsManage
	}
	return rbac.PermissionAppAccess
}
