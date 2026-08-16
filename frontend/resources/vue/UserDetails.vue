<template>
	<Section
		:title="user?.username || 'User'"
		subtitle="Account details for this user"
		classes="user-details"
	>
		<template #toolbar>
			<router-link :to="{ name: 'settingsUsers' }" class="button inline-icon neutral">
				<HugeiconsIcon
					:icon="ArrowLeft01Icon"
					width="1em"
					height="1em"
					:strokeWidth="iconStrokeWidth"
					aria-hidden="true"
				/>
				<span>Back to Users</span>
			</router-link>
			<button
				type="button"
				class="inline-icon neutral"
				aria-label="Refresh"
				title="Refresh"
				:disabled="loading"
				@click="load"
			>
				<HugeiconsIcon
					:icon="RefreshIcon"
					width="1em"
					height="1em"
					:strokeWidth="iconStrokeWidth"
					aria-hidden="true"
				/>
			</button>
			<button
				v-if="canImpersonate"
				type="button"
				class="inline-icon neutral"
				title="Login as this user"
				:disabled="impersonating"
				@click="impersonate"
			>
				<HugeiconsIcon
					:icon="LoginSquare01Icon"
					width="1em"
					height="1em"
					:strokeWidth="iconStrokeWidth"
					aria-hidden="true"
				/>
				<span>{{ impersonating ? 'Switching…' : 'Impersonate' }}</span>
			</button>
		</template>

		<div v-if="loading">
			<p>Loading...</p>
		</div>
		<div v-else-if="error">
			<p class="inline-notification error">{{ error }}</p>
		</div>
		<div v-else-if="!user">
			<p class="inline-notification note">User not found.</p>
		</div>
		<template v-else>
			<dl class="user-meta">
				<dt>User ID</dt>
				<dd>{{ user.id }}</dd>
				<dt>Username</dt>
				<dd>{{ user.username }}</dd>
				<dt>Created</dt>
				<dd>{{ user.createdAt }}</dd>
				<dt>Created by</dt>
				<dd>{{ user.createdBy || '—' }}</dd>
			</dl>

			<div v-if="showUserAdminNav" class="user-admin-nav">
				<h3 class="subsection-title">Account administration</h3>
				<Navigation ref="userSubNav">
					<NavigationGrid />
				</Navigation>
			</div>
		</template>
	</Section>

	<Section
		v-if="canViewGroups && !loading && user"
		subtitle="Groups this user belongs to. Group membership controls inherited roles and permissions."
		classes="user-details-groups settings-users"
		:padding="memberGroups.length === 0"
	>
		<template #title>
			<span class="section-title-with-icon">
				<HugeiconsIcon :icon="UserGroupIcon" width="22" height="22" aria-hidden="true" />
				User group membership for {{ user.username }}
			</span>
		</template>

		<template #toolbar>
			<button
				type="button"
				class="inline-icon neutral"
				aria-label="Refresh"
				title="Refresh"
				:disabled="groupsLoading || groupsSaving"
				@click="loadUserGroups"
			>
				<HugeiconsIcon
					:icon="RefreshIcon"
					width="1em"
					height="1em"
					:strokeWidth="iconStrokeWidth"
					aria-hidden="true"
				/>
			</button>
			<button
				v-if="canManageGroups"
				type="button"
				class="inline-icon good"
				aria-label="Add to groups"
				title="Add to groups"
				:disabled="groupsLoading || groupsSaving"
				@click="openGroupLookup"
			>
				<HugeiconsIcon
					:icon="Add01Icon"
					width="1em"
					height="1em"
					:strokeWidth="iconStrokeWidth"
					aria-hidden="true"
				/>
			</button>
		</template>

		<div v-if="groupsLoading && !memberGroups.length" class="list-banner-pad muted">Loading groups…</div>

		<template v-else>
			<p v-if="!canManageGroups" class="inline-notification note list-banner-pad">
				You can view group membership. Editing requires the usergroups.manage permission.
			</p>
			<div
				v-if="groupsSaveMessage"
				class="inline-notification list-banner-pad"
				:class="groupsSaveType"
			>{{ groupsSaveMessage }}</div>

			<p v-if="!allGroups.length" class="inline-notification note">No user groups exist yet.</p>
			<p v-else-if="!memberGroups.length" class="inline-notification note">This user is not a member of any groups.</p>

			<Table
				v-else
				class="groups-table-wrap"
				row-clickable
				:data="memberGroups"
				:headers="groupTableHeaders"
				@row-click="openGroupDetails"
			>
				<template #cell-name="{ value }">
					<strong>{{ value }}</strong>
				</template>
				<template #cell-actions="{ row }">
					<div v-if="canManageGroups" class="actions-cell">
						<button type="button" class="bad small" :disabled="groupsSaving" @click="removeFromGroup(row)">
							Remove
						</button>
					</div>
				</template>
			</Table>
		</template>
	</Section>

	<UserGroupLookupDialog
		ref="groupLookup"
		title="Add user to groups"
		:subtitle="user ? `Select groups to add ${user.username} to. Groups already joined are hidden.` : ''"
		multiple
		confirm-label="Add to groups"
		:exclude-group-ids="userGroupIds"
		@picked="onGroupsPicked"
	/>

</template>

<script setup>
	import { ref, computed, watch, nextTick } from 'vue';
	import { useRoute, useRouter } from 'vue-router';
	import { HugeiconsIcon } from '@hugeicons/vue';
	import {
		Add01Icon,
		ArrowLeft01Icon,
		LoginSquare01Icon,
		RefreshIcon,
		UserGroupIcon,
	} from '@hugeicons/core-free-icons';
	import Section from 'picocrank/vue/components/Section.vue';
	import Navigation from 'picocrank/vue/components/Navigation.vue';
	import NavigationGrid from 'picocrank/vue/components/NavigationGrid.vue';
	import Table from './picocrank/TableWithRowClick.vue';
	import UserGroupLookupDialog from './UserGroupLookupDialog.vue';
	import { waitForClient } from '../javascript/util';

	const route = useRoute();
	const router = useRouter();
	const iconStrokeWidth = 2.5;
	const user = ref(null);
	const loading = ref(true);
	const error = ref('');
	const impersonating = ref(false);

	const statusPerms = ref([]);
	const statusSuper = ref(false);

	const allGroups = ref([]);
	const userGroupIds = ref([]);
	const groupRoleCounts = ref({});
	const groupsLoading = ref(false);
	const groupsSaving = ref(false);
	const groupsSaveMessage = ref('');
	const groupsSaveType = ref('');
	const groupLookup = ref(null);

	const userSubNav = ref(null);

	const canManageRoles = computed(
		() => statusSuper.value || (Array.isArray(statusPerms.value) && statusPerms.value.includes('rbac.manage'))
	);

	const canResetPassword = computed(
		() => statusSuper.value || (Array.isArray(statusPerms.value) && statusPerms.value.includes('users.reset-password'))
	);

	const canViewApiKeys = computed(
		() => statusSuper.value || (Array.isArray(statusPerms.value) && statusPerms.value.includes('app.access'))
	);

	const canViewUserAccess = computed(
		() => canViewGroups.value || canManageRoles.value
	);

	const showUserAdminNav = computed(
		() => canViewUserAccess.value || canResetPassword.value || canViewApiKeys.value
	);

	const canImpersonate = computed(
		() => statusSuper.value || (Array.isArray(statusPerms.value) && statusPerms.value.includes('system.impersonate'))
	);

	const canViewGroups = computed(
		() => statusSuper.value || (Array.isArray(statusPerms.value) && statusPerms.value.includes('usergroups.view'))
	);

	const canManageGroups = computed(
		() => statusSuper.value || (Array.isArray(statusPerms.value) && statusPerms.value.includes('usergroups.manage'))
	);

	const canViewRoles = computed(
		() => statusSuper.value || (Array.isArray(statusPerms.value) && statusPerms.value.includes('rbac.view'))
	);

	const memberGroups = computed(() =>
		userGroupIds.value
			.map((id) => {
				const g = allGroups.value.find((g) => g.id === id);
				if (!g) return null;
				return {
					...g,
					roleCount: groupRoleCounts.value[id] ?? 0,
				};
			})
			.filter(Boolean)
			.sort((a, b) => a.name.localeCompare(b.name)),
	);

	const groupTableHeaders = computed(() => {
		const headers = [
			{ key: 'name', label: 'Group', sortable: true },
			{ key: 'memberCount', label: 'Members', sortable: true, width: '8rem' },
		];
		if (canViewRoles.value) {
			headers.push({ key: 'roleCount', label: 'Roles', sortable: true, width: '8rem' });
		}
		if (canManageGroups.value) {
			headers.push({ key: 'actions', label: 'Actions', sortable: false, width: '6rem' });
		}
		return headers;
	});

	const userId = computed(() => {
		const n = parseInt(String(route.params.id), 10);
		return Number.isFinite(n) && n > 0 ? n : 0;
	});

	function openGroupDetails(row) {
		router.push({ name: 'userGroupDetails', params: { id: String(row.id) } });
	}

	function openGroupLookup() {
		groupLookup.value?.open();
	}

	function setupUserSubNavLinks() {
		const nav = userSubNav.value;
		if (!nav || !userId.value) {
			return;
		}
		nav.clearNavigationLinks();
		if (canViewUserAccess.value) {
			nav.addRouterLink('userDetailsRoles', null, {
				params: { id: String(userId.value) },
				description: 'Manage group membership and review effective permissions',
			});
		}
		if (canResetPassword.value) {
			nav.addRouterLink('userDetailsPassword', null, {
				params: { id: String(userId.value) },
				description: 'Set a new password for this account',
			});
		}
		if (canViewApiKeys.value) {
			nav.addRouterLink('userDetailsApiKeys', null, {
				params: { id: String(userId.value) },
				description: 'View and manage API keys for this user',
			});
		}
	}

	async function refreshStatus() {
		await waitForClient();
		const st = await window.client.getStatus({});
		statusPerms.value = st.rbacPermissions || [];
		statusSuper.value = Boolean(st.rbacIsSuperuser);
	}

	async function loadUserGroups() {
		if (!canViewGroups.value || !userId.value) return;
		groupsLoading.value = true;
		groupsSaveMessage.value = '';
		groupsSaveType.value = '';
		try {
			await waitForClient();
			const groupsRes = await window.client.listUserGroups({});
			allGroups.value = groupsRes.groups || [];

			const memberPromises = allGroups.value.map((g) =>
				window.client.getUserGroupMembers({ groupId: g.id })
			);
			const memberResults = await Promise.all(memberPromises);
			const ids = [];
			for (let i = 0; i < allGroups.value.length; i++) {
				const members = memberResults[i].userIds || [];
				if (members.includes(userId.value)) {
					ids.push(allGroups.value[i].id);
				}
			}
			userGroupIds.value = ids;

			if (canViewRoles.value && ids.length) {
				const roleResults = await Promise.all(
					ids.map((id) => window.client.getUserGroupRbacRoles({ groupId: id })),
				);
				const counts = {};
				for (let i = 0; i < ids.length; i++) {
					counts[ids[i]] = (roleResults[i].roleIds || []).length;
				}
				groupRoleCounts.value = counts;
			} else {
				groupRoleCounts.value = {};
			}
		} catch (e) {
			console.error(e);
			groupsSaveMessage.value = e.message || 'Failed to load group membership.';
			groupsSaveType.value = 'error';
		} finally {
			groupsLoading.value = false;
		}
	}

	async function addUserToGroup(groupId) {
		const res = await window.client.getUserGroupMembers({ groupId });
		const current = res.userIds || [];
		if (current.includes(userId.value)) {
			return false;
		}
		await window.client.setUserGroupMembers({
			groupId,
			userIds: [...current, userId.value],
		});
		return true;
	}

	async function removeUserFromGroup(groupId) {
		const res = await window.client.getUserGroupMembers({ groupId });
		const current = res.userIds || [];
		if (!current.includes(userId.value)) {
			return false;
		}
		await window.client.setUserGroupMembers({
			groupId,
			userIds: current.filter((id) => id !== userId.value),
		});
		return true;
	}

	async function onGroupsPicked(groups) {
		if (!canManageGroups.value || !groups.length) return;
		groupsSaving.value = true;
		groupsSaveMessage.value = '';
		groupsSaveType.value = '';
		try {
			await waitForClient();
			let added = 0;
			for (const group of groups) {
				if (await addUserToGroup(group.id)) {
					added++;
				}
			}
			if (added === 0) {
				groupsSaveMessage.value = 'No new groups selected.';
				groupsSaveType.value = 'note';
			} else {
				groupsSaveMessage.value = added === 1
					? `Added user to ${groups[0].name}.`
					: `Added user to ${added} groups.`;
				groupsSaveType.value = 'success';
			}
			await loadUserGroups();
		} catch (e) {
			groupsSaveMessage.value = e.message || 'Failed to update group membership.';
			groupsSaveType.value = 'error';
		} finally {
			groupsSaving.value = false;
		}
	}

	async function removeFromGroup(group) {
		if (!canManageGroups.value || !confirm(`Remove "${user.value?.username || 'this user'}" from ${group.name}?`)) {
			return;
		}
		groupsSaving.value = true;
		groupsSaveMessage.value = '';
		groupsSaveType.value = '';
		try {
			await waitForClient();
			await removeUserFromGroup(group.id);
			groupsSaveMessage.value = `Removed user from ${group.name}.`;
			groupsSaveType.value = 'success';
			await loadUserGroups();
		} catch (e) {
			groupsSaveMessage.value = e.message || 'Failed to update group membership.';
			groupsSaveType.value = 'error';
		} finally {
			groupsSaving.value = false;
		}
	}

	async function load() {
		if (!userId.value) {
			user.value = null;
			error.value = 'Invalid user ID.';
			loading.value = false;
			return;
		}

		loading.value = true;
		error.value = '';
		user.value = null;

		try {
			await waitForClient();
			await refreshStatus();
			const res = await window.client.getUser({ userId: userId.value });
			user.value = res.user ?? null;
			if (!user.value) {
				error.value = 'User not found.';
			}
		} catch (e) {
			error.value = e.message || 'Failed to load user.';
			user.value = null;
		} finally {
			loading.value = false;
		}

		await loadUserGroups();

		await nextTick();
		await nextTick();
		setupUserSubNavLinks();
	}

	async function impersonate() {
		if (!userId.value || !confirm(`Impersonate ${user.value?.username || 'this user'}? You will see the app as they do.`)) return;
		impersonating.value = true;
		try {
			await waitForClient();
			const res = await window.client.impersonateUser({ userId: userId.value });
			if (res.standardResponse?.success) {
				window.location.href = '/';
			}
		} catch (e) {
			alert('Failed to impersonate: ' + (e.message || e));
		} finally {
			impersonating.value = false;
		}
	}

	watch(userId, () => {
		load();
	}, { immediate: true });

	watch([showUserAdminNav, canViewUserAccess, canResetPassword, canViewApiKeys, userId, loading], () => {
		if (loading.value || !user.value || !showUserAdminNav.value) {
			return;
		}
		nextTick(() => {
			nextTick(() => setupUserSubNavLinks());
		});
	}, { flush: 'post' });
</script>

<style scoped>
	.user-meta {
		display: grid;
		grid-template-columns: minmax(6rem, 10rem) 1fr;
		gap: 0.35rem 1.25rem;
		margin: 0 0 1.5rem;
	}

	.user-meta dt {
		margin: 0;
		font-weight: 600;
		opacity: 0.85;
	}

	.user-meta dd {
		margin: 0;
	}

	.user-admin-nav {
		margin-bottom: 1.5rem;
	}

	.subsection-title {
		margin: 0 0 0.5rem;
		font-size: 1rem;
		font-weight: 600;
	}

	.section-title-with-icon {
		display: inline-flex;
		align-items: center;
		gap: 0.45em;
		vertical-align: middle;
	}

	.list-banner-pad {
		padding-left: 1em;
		padding-right: 1em;
	}

	.groups-table-wrap {
		margin-top: 0.5rem;
		margin-bottom: 1.5rem;
	}

	.actions-cell {
		text-align: right;
	}
</style>
