package authentication

import (
	controlv1 "github.com/jamesread/japella/gen/japella/controlapi/v1/controlv1connect"
	"github.com/jamesread/japella/internal/rbac"
)

// procedurePermissions maps Connect procedures that need more than app.access.
// Allow-listed unauthenticated procedures are handled before RequiredPermission runs.
// GetUsers lists id + username for all accounts (pickers, sharing, groups) and stays
// on the app.access default; sensitive per-user operations still use GetUser and users.view.
var procedurePermissions = map[string]string{
	controlv1.JapellaControlApiServiceGetUserProcedure:                    rbac.PermissionUsersView,
	controlv1.JapellaControlApiServiceCreateUserProcedure:                 rbac.PermissionUsersCreate,
	controlv1.JapellaControlApiServiceDeleteUserProcedure:                 rbac.PermissionUsersDelete,
	controlv1.JapellaControlApiServiceResetUserPasswordProcedure:          rbac.PermissionUsersResetPassword,
	controlv1.JapellaControlApiServiceListRbacPermissionsProcedure:        rbac.PermissionRbacView,
	controlv1.JapellaControlApiServiceListRbacRolesProcedure:              rbac.PermissionRbacView,
	controlv1.JapellaControlApiServiceGetUserRbacRolesProcedure:           rbac.PermissionRbacView,
	controlv1.JapellaControlApiServiceGetUserGroupRbacRolesProcedure:      rbac.PermissionRbacView,
	controlv1.JapellaControlApiServiceGetRbacRoleUsersProcedure:           rbac.PermissionRbacView,
	controlv1.JapellaControlApiServiceGetRbacRoleGroupsProcedure:          rbac.PermissionRbacView,
	controlv1.JapellaControlApiServiceCreateRbacRoleProcedure:             rbac.PermissionRbacManage,
	controlv1.JapellaControlApiServiceUpdateRbacRoleProcedure:             rbac.PermissionRbacManage,
	controlv1.JapellaControlApiServiceDeleteRbacRoleProcedure:             rbac.PermissionRbacManage,
	controlv1.JapellaControlApiServiceSetUserGroupRbacRolesProcedure:      rbac.PermissionRbacManage,
	controlv1.JapellaControlApiServiceGetCvarsProcedure:                   rbac.PermissionSystemSettings,
	controlv1.JapellaControlApiServiceSetCvarProcedure:                    rbac.PermissionSystemSettings,
	controlv1.JapellaControlApiServiceStopServiceProcedure:                rbac.PermissionSystemSettings,
	controlv1.JapellaControlApiServiceListWebhooksProcedure:               rbac.PermissionSystemSettings,
	controlv1.JapellaControlApiServiceGetWebhookProcedure:                 rbac.PermissionSystemSettings,
	controlv1.JapellaControlApiServiceCreateWebhookProcedure:              rbac.PermissionSystemSettings,
	controlv1.JapellaControlApiServiceUpdateWebhookProcedure:              rbac.PermissionSystemSettings,
	controlv1.JapellaControlApiServiceDeleteWebhookProcedure:              rbac.PermissionSystemSettings,
	controlv1.JapellaControlApiServiceTestWebhookProcedure:                rbac.PermissionSystemSettings,
	controlv1.JapellaControlApiServiceListBotWebhooksProcedure:            rbac.PermissionSystemSettings,
	controlv1.JapellaControlApiServiceListLlmConnectionsProcedure:         rbac.PermissionSystemSettings,
	controlv1.JapellaControlApiServiceGetLlmConnectionProcedure:           rbac.PermissionSystemSettings,
	controlv1.JapellaControlApiServiceCreateLlmConnectionProcedure:        rbac.PermissionSystemSettings,
	controlv1.JapellaControlApiServiceUpdateLlmConnectionProcedure:        rbac.PermissionSystemSettings,
	controlv1.JapellaControlApiServiceDeleteLlmConnectionProcedure:        rbac.PermissionSystemSettings,
	controlv1.JapellaControlApiServiceTestLlmConnectionProcedure:          rbac.PermissionSystemSettings,
	controlv1.JapellaControlApiServiceListLlmConnectionPresetsProcedure:   rbac.PermissionSystemSettings,
	controlv1.JapellaControlApiServiceListLlmConnectionModelsProcedure:    rbac.PermissionSystemSettings,
	controlv1.JapellaControlApiServiceListLlmToolsProcedure:               rbac.PermissionSystemSettings,
	controlv1.JapellaControlApiServiceGetLlmToolProcedure:                 rbac.PermissionSystemSettings,
	controlv1.JapellaControlApiServiceCreateLlmToolProcedure:              rbac.PermissionSystemSettings,
	controlv1.JapellaControlApiServiceUpdateLlmToolProcedure:              rbac.PermissionSystemSettings,
	controlv1.JapellaControlApiServiceDeleteLlmToolProcedure:              rbac.PermissionSystemSettings,
	controlv1.JapellaControlApiServiceListHealthIssuesProcedure:           rbac.PermissionSystemSettings,
	controlv1.JapellaControlApiServiceGetHealthSummaryProcedure:           rbac.PermissionSystemSettings,
	controlv1.JapellaControlApiServiceRunHealthChecksProcedure:            rbac.PermissionSystemSettings,
	controlv1.JapellaControlApiServiceAcknowledgeHealthIssueProcedure:     rbac.PermissionSystemSettings,
	controlv1.JapellaControlApiServiceGetConnectorsProcedure:              rbac.PermissionSystemConnectors,
	controlv1.JapellaControlApiServiceRefreshConnectorsProcedure:          rbac.PermissionSystemConnectors,
	controlv1.JapellaControlApiServiceStartOAuthProcedure:                 rbac.PermissionSystemConnectors,
	controlv1.JapellaControlApiServiceRegisterConnectorProcedure:          rbac.PermissionSystemConnectors,
	controlv1.JapellaControlApiServiceGetJobsStatusProcedure:              rbac.PermissionSystemLogs,
	controlv1.JapellaControlApiServiceCleanupFeedPostsProcedure:           rbac.PermissionSystemLogs,
	controlv1.JapellaControlApiServiceImpersonateUserProcedure:            rbac.PermissionSystemImpersonate,
	controlv1.JapellaControlApiServiceStopImpersonationProcedure:          "",
	controlv1.JapellaControlApiServiceGetUserGroupMembersProcedure:        rbac.PermissionUserGroupsView,
	controlv1.JapellaControlApiServiceGetUserGroupSharedAccountsProcedure: rbac.PermissionUserGroupsView,
	controlv1.JapellaControlApiServiceCreateUserGroupProcedure:            rbac.PermissionUserGroupsManage,
	controlv1.JapellaControlApiServiceDeleteUserGroupProcedure:            rbac.PermissionUserGroupsManage,
	controlv1.JapellaControlApiServiceSetUserGroupMembersProcedure:        rbac.PermissionUserGroupsManage,
	controlv1.JapellaControlApiServiceListAccountPoliciesProcedure:        rbac.PermissionAccountPoliciesManage,
	controlv1.JapellaControlApiServiceGetAccountPolicyProcedure:           rbac.PermissionAccountPoliciesManage,
	controlv1.JapellaControlApiServiceCreateAccountPolicyProcedure:        rbac.PermissionAccountPoliciesManage,
	controlv1.JapellaControlApiServiceUpdateAccountPolicyProcedure:        rbac.PermissionAccountPoliciesManage,
	controlv1.JapellaControlApiServiceDeleteAccountPolicyProcedure:        rbac.PermissionAccountPoliciesManage,
}

// RequiredPermission returns the permission needed for an authenticated Connect procedure.
// Allow-listed unauthenticated procedures are handled before this runs.
// Default is app.access for normal application use.
func RequiredPermission(procedureName string) string {
	if p, ok := procedurePermissions[procedureName]; ok {
		return p
	}
	return rbac.PermissionAppAccess
}
