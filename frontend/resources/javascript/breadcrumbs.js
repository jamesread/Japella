/**
 * Breadcrumb trails for picocrank Header breadcrumbs (meta.breadcrumbs).
 * Each route meta.breadcrumbs(route) returns the full trail for that page.
 */

/** @typedef {{ name: string, href: string }} BreadcrumbItem */

/** @param {...BreadcrumbItem} items @returns {BreadcrumbItem[]} */
export function trail(...items) {
	return items
}

/** @param {import('vue-router').RouteLocationNormalizedLoaded} route @param {string} title */
export function currentPage(route, title) {
	return { name: title, href: route.fullPath }
}

export const crumb = {
	controlPanel: { name: 'Control Panel', href: '/control-panel' },
	iam: { name: 'IAM', href: '/control-panel/iam' },
	users: { name: 'Users', href: '/users' },
	userGroups: { name: 'User Groups', href: '/user-groups' },
	accountPolicies: { name: 'Account Policies', href: '/account-policies' },
	rbac: { name: 'Roles & permissions', href: '/settings/rbac' },
	permissions: { name: 'Permissions', href: '/settings/rbac/permissions' },
	connectors: { name: 'Connectors', href: '/connectors' },
	systemSettings: { name: 'System Settings', href: '/settings' },
	webhooks: { name: 'Webhooks', href: '/admin/webhooks' },
	systemLogs: { name: 'System Logs', href: '/logs' },
	systemArchitecture: { name: 'System Architecture', href: '/control-panel/system-architecture' },
	systemDiagnostics: { name: 'System Diagnostics', href: '/control-panel/system-diagnostics' },
	socialAccounts: { name: 'Social Accounts', href: '/social-accounts' },
	chatBots: { name: 'Chat Bots', href: '/chat-bots' },
	campaigns: { name: 'Campaigns', href: '/campaigns' },
	timeline: { name: 'Timeline', href: '/timeline' },
}

/** @param {...BreadcrumbItem} tail */
export function controlPanelTrail(...tail) {
	return trail(crumb.controlPanel, ...tail)
}

/** @param {...BreadcrumbItem} tail */
export function iamTrail(...tail) {
	return trail(crumb.controlPanel, crumb.iam, ...tail)
}

/** @param {string} name @param {import('vue-router').RouteLocationNormalizedLoaded} route */
function userCrumb(route, name = 'User') {
	return { name, href: `/users/${route.params.id}` }
}

/**
 * Breadcrumb builders keyed by route name.
 * @type {Record<string, (route: import('vue-router').RouteLocationNormalizedLoaded) => BreadcrumbItem[]>}
 */
export const breadcrumbsByRouteName = {
	controlPanel: () => trail(crumb.controlPanel),
	iam: () => iamTrail(),
	connectors: () => controlPanelTrail(crumb.connectors),
	settings: () => controlPanelTrail(crumb.systemSettings),
	adminWebhooks: () => controlPanelTrail(crumb.webhooks),
	logs: () => controlPanelTrail(crumb.systemLogs),
	systemArchitecture: () => controlPanelTrail(crumb.systemArchitecture),
	systemDiagnostics: () => controlPanelTrail(crumb.systemDiagnostics),

	settingsUsers: () => iamTrail(crumb.users),
	createUser: (route) => iamTrail(crumb.users, currentPage(route, 'Create User')),
	userDetails: (route) => iamTrail(crumb.users, userCrumb(route)),
	userDetailsRoles: (route) => iamTrail(crumb.users, userCrumb(route), currentPage(route, 'Permissions Audit')),
	userDetailsPassword: (route) => iamTrail(crumb.users, userCrumb(route), currentPage(route, 'Reset password')),
	userDetailsApiKeys: (route) => iamTrail(crumb.users, userCrumb(route), currentPage(route, 'API Keys')),

	userGroups: () => iamTrail(crumb.userGroups),
	userGroupDetails: (route) => iamTrail(crumb.userGroups, currentPage(route, 'User Group')),

	accountPolicies: () => iamTrail(crumb.accountPolicies),
	createAccountPolicy: (route) => iamTrail(crumb.accountPolicies, currentPage(route, 'Create Account Policy')),
	editAccountPolicy: (route) => iamTrail(crumb.accountPolicies, currentPage(route, 'Edit Account Policy')),

	rbacSettings: () => iamTrail(crumb.rbac),
	rbacPermissions: () => iamTrail(crumb.permissions),
	rbacRoleDetails: (route) => iamTrail(crumb.rbac, currentPage(route, 'Role')),

	socialAccounts: () => trail(crumb.socialAccounts),
	addSocialAccount: (route) => trail(crumb.socialAccounts, currentPage(route, 'Add Social Account')),
	socialAccountPreAdd: (route) => trail(crumb.socialAccounts, currentPage(route, 'Connect Account')),
	socialAccountDetails: (route) => trail(crumb.socialAccounts, currentPage(route, 'Social Account')),

	chatBots: () => trail(crumb.chatBots),
	createChatBot: (route) => trail(crumb.chatBots, currentPage(route, 'Add Chat Bot')),
	chatBotDetails: (route) => trail(crumb.chatBots, currentPage(route, 'Chat Bot Details')),
	chatBotConversations: (route) => trail(crumb.chatBots, currentPage(route, 'Bot Conversations')),
	chatBotHooks: (route) => trail(crumb.chatBots, currentPage(route, 'Bot Message Hooks')),
	chatBotConversationsAll: (route) => trail(currentPage(route, 'Conversations')),

	postBox: (route) => trail(currentPage(route, 'Post')),
	approvals: (route) => trail(currentPage(route, 'Approvals')),
	media: (route) => trail(currentPage(route, 'Media')),
	campaigns: () => trail(crumb.campaigns),
	campaignDetails: (route) => trail(crumb.campaigns, currentPage(route, 'Campaign Details')),
	cannedPosts: (route) => trail(currentPage(route, 'Canned Posts')),
	timeline: () => trail(crumb.timeline),
	postDetails: (route) => trail(crumb.timeline, currentPage(route, 'Post Details')),
	feed: (route) => trail(currentPage(route, 'Feed')),
	calendar: (route) => trail(currentPage(route, 'Calendar')),

	settingsApiKeys: (route) => trail(currentPage(route, 'API Keys')),
	userControlPanel: (route) => trail(currentPage(route, 'User Control Panel')),
	myPermissions: (route) => trail(
		{ name: 'User Control Panel', href: '/user-control-panel' },
		currentPage(route, 'My Permissions'),
	),
	userPreferences: (route) => trail(
		{ name: 'User Control Panel', href: '/user-control-panel' },
		currentPage(route, 'User Preferences'),
	),
	changePassword: (route) => trail(currentPage(route, 'Change Password')),
	browserDiagnostics: (route) => trail(currentPage(route, 'Browser Diagnostics')),
	welcome: () => trail({ name: 'Welcome', href: '/' }),
}

/**
 * @param {import('vue-router').RouteRecordRaw} route
 */
export function applyRouteBreadcrumbs(route) {
	if (!route.meta) {
		route.meta = {}
	}
	const builder = breadcrumbsByRouteName[route.name]
	if (builder) {
		route.meta.breadcrumbs = builder
	} else if (route.meta.title && route.name) {
		route.meta.breadcrumbs = (r) => [currentPage(r, route.meta.title)]
	}
}
