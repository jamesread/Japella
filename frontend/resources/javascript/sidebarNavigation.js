import { canAccessControlPanelFromStatus } from './rbacAccess.js';

export const SIDEBAR_NAV_SECTIONS = [
	{
		title: 'Write',
		name: 'nav-write',
		routes: ['postBox', 'approvals', 'media', 'campaigns', 'cannedPosts', 'calendar', 'timeline'],
	},
	{
		title: 'Read',
		name: 'nav-read',
		routes: ['feed', 'chatBotConversationsAll'],
	},
	{
		title: 'Settings',
		name: 'nav-settings',
		routes: ['socialAccounts', 'chatBots', 'controlPanel'],
	},
];

export async function fetchApprovalsCount() {
	try {
		const res = await window.client.listPendingApprovals({});
		return (res.pending || []).length;
	} catch (e) {
		console.warn('Failed to load approvals count for navigation', e);
		return 0;
	}
}

/**
 * @param {import('vue').Ref | { clearNavigationLinks: Function, addSection: Function, addRouterLink: Function }} navigation
 * @param {{ approvalsCount?: number, showControlPanel?: boolean, descriptions?: Record<string, string>, excludeRoutes?: string[] }} [options]
 */
export function setupSidebarNavigation(navigation, options = {}) {
	const {
		approvalsCount = 0,
		showControlPanel = false,
		descriptions = {},
		excludeRoutes = [],
	} = options;

	const excluded = new Set(excludeRoutes);

	navigation.clearNavigationLinks();

	for (const section of SIDEBAR_NAV_SECTIONS) {
		navigation.addSection(section.title, { name: section.name });
		for (const routeName of section.routes) {
			if (excluded.has(routeName)) {
				continue;
			}
			if (routeName === 'controlPanel' && !showControlPanel) {
				continue;
			}
			const linkOptions = {};
			if (routeName === 'approvals') {
				linkOptions.count = approvalsCount;
			}
			if (descriptions[routeName]) {
				linkOptions.description = descriptions[routeName];
			}
			navigation.addRouterLink(routeName, null, linkOptions);
		}
	}
}

export function createSectionLinkFilter(section) {
	const routeNames = new Set(section.routes);
	return (link) => routeNames.has(link.name);
}

export function showControlPanelInSidebar(status) {
	return canAccessControlPanelFromStatus({
		isLoggedIn: true,
		rbacIsSuperuser: status?.rbacIsSuperuser ?? window.userRbacIsSuperuser,
		rbacPermissions: status?.rbacPermissions ?? window.userRbacPermissions ?? [],
	});
}
