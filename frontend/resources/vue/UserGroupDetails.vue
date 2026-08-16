<template>
	<Section
		subtitle="Group overview."
		classes="user-group-details"
	>
		<template #title>
			<span class="section-title-with-icon">
				<HugeiconsIcon :icon="UserGroupIcon" width="22" height="22" aria-hidden="true" />
				{{ group?.name || 'User group' }}
			</span>
		</template>
		<template #toolbar>
			<router-link :to="{ name: 'userGroups' }" class="button inline-icon neutral">
				<HugeiconsIcon
					:icon="ArrowLeft01Icon"
					width="1em"
					height="1em"
					:strokeWidth="iconStrokeWidth"
					aria-hidden="true"
				/>
				<span>Back to User Groups</span>
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
				v-if="canManage && group && !isSystemGroup"
				type="button"
				class="inline-icon bad"
				:disabled="saving"
				@click="deleteGroup"
			>
				<HugeiconsIcon
					:icon="Delete02Icon"
					width="1em"
					height="1em"
					:strokeWidth="iconStrokeWidth"
					aria-hidden="true"
				/>
				<span>Delete group</span>
			</button>
		</template>

		<div v-if="errorMessage" class="inline-notification error">{{ errorMessage }}</div>
		<div v-if="actionMessage" class="inline-notification" :class="actionMessageType">{{ actionMessage }}</div>

		<div v-if="loading" class="muted">Loading…</div>
		<div v-else-if="!group" class="inline-notification note">User group not found.</div>
		<template v-else>
			<dl class="group-meta">
				<dt>Group ID</dt>
				<dd>{{ group.id }}</dd>
				<dt>Name</dt>
				<dd>{{ group.name }}</dd>
				<dt>Members</dt>
				<dd>{{ members.length }}</dd>
				<dt>RBAC roles</dt>
				<dd>{{ assignedRoleRows.length }}</dd>
				<dt>Shared accounts</dt>
				<dd>{{ sharedAccounts.length }}</dd>
			</dl>
		</template>
	</Section>

	<Section
		v-if="!loading && group && canViewRoles"
		subtitle="Members of this group inherit these roles and their permissions."
		classes="user-group-details-roles settings-users"
		:padding="assignedRoleRows.length === 0"
	>
		<template #title>
			<span class="section-title-with-icon">
				<HugeiconsIcon :icon="WebSecurityIcon" width="22" height="22" aria-hidden="true" />
				RBAC roles
			</span>
		</template>

		<template #toolbar>
			<button
				type="button"
				class="inline-icon neutral"
				aria-label="Refresh"
				title="Refresh"
				:disabled="loading || rolesSaving"
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
				v-if="canManageRoles && availableRoleOptions.length"
				type="button"
				class="inline-icon good"
				aria-label="Add roles"
				title="Add roles"
				:disabled="rolesSaving"
				@click="openAddRolesDialog"
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

		<p v-if="isSystemGroup" class="inline-notification note list-banner-pad">
			<strong>{{ group.name }}</strong> is a system group. Its roles are managed carefully — removing
			<span v-if="group.name === 'Everyone'">the member role</span>
			<span v-else-if="group.name === 'Administrators'">the superuser role</span>
			may lock out users.
		</p>
		<p v-if="!canManageRoles" class="inline-notification note list-banner-pad">
			You can view group roles. Editing requires the rbac.manage permission.
		</p>
		<div
			v-if="rolesActionMessage"
			class="inline-notification list-banner-pad"
			:class="rolesActionMessageType"
		>{{ rolesActionMessage }}</div>

		<p v-if="!assignedRoleRows.length" class="inline-notification note">No roles assigned to this group.</p>

		<Table
			v-else
			class="roles-table-wrap"
			row-clickable
			:data="assignedRoleRows"
			:headers="roleTableHeaders"
			@row-click="openRoleDetails"
		>
			<template #cell-name="{ row, value }">
				<router-link
					:to="{ name: 'rbacRoleDetails', params: { id: String(row.id) } }"
					class="username-link"
					@click.stop
				>
					<strong>{{ value }}</strong>
				</router-link>
			</template>
			<template #cell-tags="{ row }">
				<span class="role-tags">
					<span v-if="isSystemRoleName(row.name)" class="tag fg-note">system role</span>
					<span v-if="row.name === 'superuser'" class="tag fg-good">all access</span>
					<span v-if="!isSystemRoleName(row.name)" class="muted">—</span>
				</span>
			</template>
			<template #cell-actions="{ row }">
				<div v-if="canManageRoles" class="actions-cell">
					<button type="button" class="bad small" :disabled="rolesSaving" @click="revokeRole(row)">
						Revoke
					</button>
				</div>
			</template>
		</Table>
	</Section>

	<Section
		v-if="!loading && group"
		subtitle="Social accounts shared with this group. Members inherit the listed permissions."
		classes="user-group-details-shared-accounts"
	>
		<template #title>
			<span class="section-title-with-icon">
				<HugeiconsIcon :icon="LinkSquare01Icon" width="22" height="22" aria-hidden="true" />
				Shared social accounts
			</span>
		</template>
		<p v-if="!sharedAccounts.length" class="inline-notification note">No social accounts are shared with this group.</p>
		<table v-else class="shared-accounts-table">
			<thead>
				<tr>
					<th>Account</th>
					<th>Connector</th>
					<th class="perm-col">Read</th>
					<th class="perm-col">Post</th>
					<th class="perm-col">Manage</th>
					<th>Status</th>
				</tr>
			</thead>
			<tbody>
				<tr v-for="a in sharedAccounts" :key="a.socialAccountId">
					<td>
						<router-link
							:to="{ name: 'socialAccountDetails', params: { id: String(a.socialAccountId) } }"
							class="username-link account-link"
						>
							<Icon v-if="a.icon" :icon="a.icon" width="18" height="18" />
							{{ a.identity }}
						</router-link>
					</td>
					<td>{{ a.connector }}</td>
					<td class="perm-col">
						<Icon
							:icon="a.canRead ? 'material-symbols:check' : 'material-symbols:close'"
							width="18"
							height="18"
							:class="a.canRead ? 'perm-yes' : 'perm-no'"
						/>
					</td>
					<td class="perm-col">
						<Icon
							:icon="a.canPost ? 'material-symbols:check' : 'material-symbols:close'"
							width="18"
							height="18"
							:class="a.canPost ? 'perm-yes' : 'perm-no'"
						/>
					</td>
					<td class="perm-col">
						<Icon
							:icon="a.canManage ? 'material-symbols:check' : 'material-symbols:close'"
							width="18"
							height="18"
							:class="a.canManage ? 'perm-yes' : 'perm-no'"
						/>
					</td>
					<td>{{ a.active ? 'Active' : 'Inactive' }}</td>
				</tr>
			</tbody>
		</table>
	</Section>

	<Section
		v-if="!loading && group"
		subtitle="Users who belong to this group."
		classes="user-group-details-members settings-users"
		:padding="members.length === 0"
	>
		<template #title>
			<span class="section-title-with-icon">
				<HugeiconsIcon :icon="UserMultiple02Icon" width="22" height="22" aria-hidden="true" />
				Members
			</span>
		</template>
		<template #toolbar>
			<button
				type="button"
				class="inline-icon neutral"
				aria-label="Refresh"
				title="Refresh"
				:disabled="loading || saving"
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
				v-if="canManage"
				type="button"
				class="inline-icon good"
				aria-label="Add users"
				title="Add users"
				:disabled="saving"
				@click="openMemberLookup"
			>
				<HugeiconsIcon
					:icon="UserAdd01Icon"
					width="1em"
					height="1em"
					:strokeWidth="iconStrokeWidth"
					aria-hidden="true"
				/>
			</button>
		</template>

		<p v-if="!members.length" class="inline-notification note">No members in this group yet.</p>

		<Table
			v-else
			class="members-table-wrap"
			row-clickable
			:data="members"
			:headers="memberTableHeaders"
			@row-click="openMemberDetails"
		>
			<template #cell-username="{ row, value }">
				<router-link
					:to="{ name: 'userDetails', params: { id: String(row.id) } }"
					class="username-link"
				>
					<strong>{{ value }}</strong>
				</router-link>
			</template>
			<template #cell-actions="{ row }">
				<div v-if="canManage" class="actions-cell">
					<button type="button" class="bad small" :disabled="saving" @click="removeMember(row)">
						Remove
					</button>
				</div>
			</template>
		</Table>
	</Section>

	<UserLookupDialog
		ref="memberLookup"
		title="Add users to group"
		:subtitle="group ? `Select users to add to ${group.name}. Existing members are hidden.` : ''"
		multiple
		confirm-label="Add to group"
		:exclude-user-ids="memberIds"
		@picked="onMembersPicked"
	/>

	<dialog ref="addRolesDialog" class="dialog" @close="closeAddRolesDialog">
		<h2>Add roles</h2>
		<p>Select roles to assign to this group. Members inherit permissions from these roles.</p>

		<FormLayout @submit.prevent="submitAddRoles">
			<FormField label="Roles" fake :disabled="rolesSaving">
				<CheckGroup
					v-model="addRoleIds"
					name="user-group-add-roles"
					:options="availableRoleOptions"
					:disabled="rolesSaving"
				/>
			</FormField>

			<p v-if="addRolesError" class="inline-notification error">{{ addRolesError }}</p>

			<template #actions>
				<button type="button" class="neutral" :disabled="rolesSaving" @click="closeAddRolesDialog">Cancel</button>
				<button type="submit" class="good" :disabled="rolesSaving || !addRoleIds.length">
					{{ rolesSaving ? 'Adding…' : 'Add roles' }}
				</button>
			</template>
		</FormLayout>
	</dialog>
</template>

<script setup>
	import { ref, computed, watch } from 'vue';
	import { useRoute, useRouter } from 'vue-router';
	import { Icon } from '@iconify/vue';
	import { HugeiconsIcon } from '@hugeicons/vue';
	import {
		Add01Icon,
		ArrowLeft01Icon,
		Delete02Icon,
		LinkSquare01Icon,
		RefreshIcon,
		UserAdd01Icon,
		UserGroupIcon,
		UserMultiple02Icon,
		WebSecurityIcon,
	} from '@hugeicons/core-free-icons';
	import Section from 'picocrank/vue/components/Section.vue';
	import FormLayout from 'picocrank/vue/components/FormLayout.vue';
	import FormField from 'picocrank/vue/components/FormField.vue';
	import CheckGroup from 'picocrank/vue/components/CheckGroup.vue';
	import Table from './picocrank/TableWithRowClick.vue';
	import UserLookupDialog from './UserLookupDialog.vue';
	import { waitForClient } from '../javascript/util';

	const iconStrokeWidth = 2.5;
	const route = useRoute();
	const router = useRouter();

	const group = ref(null);
	const members = ref([]);
	const sharedAccounts = ref([]);
	const roles = ref([]);
	const selectedRoleIds = ref([]);
	const rolesSaving = ref(false);
	const rolesActionMessage = ref('');
	const rolesActionMessageType = ref('');
	const addRolesDialog = ref(null);
	const addRoleIds = ref([]);
	const addRolesError = ref('');
	const loading = ref(true);
	const saving = ref(false);
	const errorMessage = ref('');
	const actionMessage = ref('');
	const actionMessageType = ref('');
	const statusPerms = ref([]);
	const statusSuper = ref(false);
	const memberLookup = ref(null);

	const groupId = computed(() => {
		const n = parseInt(String(route.params.id), 10);
		return Number.isFinite(n) && n > 0 ? n : 0;
	});

	const canView = computed(
		() => statusSuper.value || (Array.isArray(statusPerms.value) && statusPerms.value.includes('usergroups.view'))
	);
	const canManage = computed(
		() => statusSuper.value || (Array.isArray(statusPerms.value) && statusPerms.value.includes('usergroups.manage'))
	);
	const canManageRoles = computed(
		() => statusSuper.value || (Array.isArray(statusPerms.value) && statusPerms.value.includes('rbac.manage'))
	);
	const canViewRoles = computed(
		() => statusSuper.value || (Array.isArray(statusPerms.value) && statusPerms.value.includes('rbac.view'))
	);

	const isSystemGroup = computed(
		() => group.value?.name === 'Everyone' || group.value?.name === 'Administrators'
	);

	const assignedRoleRows = computed(() =>
		selectedRoleIds.value
			.map((id) => roles.value.find((r) => r.id === id))
			.filter(Boolean)
			.map((r) => ({
				...r,
				description: r.description || '—',
			}))
			.sort((a, b) => a.name.localeCompare(b.name)),
	);

	const availableRoleOptions = computed(() =>
		roles.value
			.filter((r) => !selectedRoleIds.value.includes(r.id))
			.map((r) => ({
				value: r.id,
				label: r.description ? `${r.name} — ${r.description}` : r.name,
			})),
	);

	const roleTableHeaders = computed(() => {
		const headers = [
			{ key: 'name', label: 'Role', sortable: true },
			{ key: 'description', label: 'Description', sortable: true },
			{ key: 'tags', label: 'Tags', sortable: false, width: '11rem' },
		];
		if (canManageRoles.value) {
			headers.push({ key: 'actions', label: 'Actions', sortable: false, width: '6rem' });
		}
		return headers;
	});

	function isSystemRoleName(name) {
		return name === 'superuser' || name === 'member';
	}

	function openRoleDetails(row) {
		router.push({ name: 'rbacRoleDetails', params: { id: String(row.id) } });
	}

	function openAddRolesDialog() {
		addRoleIds.value = [];
		addRolesError.value = '';
		addRolesDialog.value?.showModal();
	}

	function closeAddRolesDialog() {
		addRolesDialog.value?.close();
		addRoleIds.value = [];
		addRolesError.value = '';
	}

	const memberIds = computed(() => members.value.map((m) => m.id));

	const memberTableHeaders = computed(() => {
		const headers = [
			{ key: 'username', label: 'Username', sortable: true },
			{ key: 'id', label: 'User ID', sortable: true, width: '8rem' },
		];
		if (canManage.value) {
			headers.push({ key: 'actions', label: 'Actions', sortable: false, width: '6rem' });
		}
		return headers;
	});

	function openMemberDetails(row) {
		router.push({ name: 'userDetails', params: { id: String(row.id) } });
	}

	async function refreshStatus() {
		await waitForClient();
		const st = await window.client.getStatus({});
		statusPerms.value = st.rbacPermissions || [];
		statusSuper.value = Boolean(st.rbacIsSuperuser);
	}

	async function load() {
		errorMessage.value = '';
		actionMessage.value = '';
		rolesActionMessage.value = '';
		rolesActionMessageType.value = '';
		loading.value = true;
		group.value = null;
		members.value = [];
		sharedAccounts.value = [];
		roles.value = [];
		selectedRoleIds.value = [];

		if (!groupId.value) {
			errorMessage.value = 'Invalid group ID.';
			loading.value = false;
			return;
		}

		try {
			await waitForClient();
			await refreshStatus();
			if (!canView.value) {
				errorMessage.value = 'You do not have permission to view user groups (usergroups.view).';
				return;
			}

			const [gr, ur, mr, sr, rolesRes, groupRolesRes] = await Promise.all([
				window.client.listUserGroups({}),
				window.client.getUsers({}),
				window.client.getUserGroupMembers({ groupId: groupId.value }),
				window.client.getUserGroupSharedAccounts({ groupId: groupId.value }),
				canViewRoles.value ? window.client.listRbacRoles({}) : Promise.resolve({ roles: [] }),
				canViewRoles.value ? window.client.getUserGroupRbacRoles({ groupId: groupId.value }) : Promise.resolve({ roleIds: [] }),
			]);

			const found = (gr.groups || []).find((g) => g.id === groupId.value) || null;
			group.value = found;

			const map = new Map();
			for (const u of ur.users || []) {
				map.set(u.id, u);
			}

			const ids = mr.userIds || [];
			members.value = ids.map((id) => {
				const u = map.get(id);
				return u || { id, username: `User #${id}` };
			}).sort((a, b) => a.username.localeCompare(b.username));

			sharedAccounts.value = sr.accounts || [];
			roles.value = rolesRes.roles || [];
			selectedRoleIds.value = [...(groupRolesRes.roleIds || [])];
		} catch (e) {
			console.error(e);
			errorMessage.value = e.message || 'Failed to load user group.';
		} finally {
			loading.value = false;
		}
	}

	async function persistGroupRoles(nextIds, successMsg) {
		if (!canManageRoles.value || !groupId.value) return;
		rolesSaving.value = true;
		rolesActionMessage.value = '';
		rolesActionMessageType.value = '';
		try {
			await waitForClient();
			await window.client.setUserGroupRbacRoles({
				groupId: groupId.value,
				roleIds: nextIds,
			});
			rolesActionMessage.value = successMsg || 'Group roles updated.';
			rolesActionMessageType.value = 'success';
			await load();
		} catch (e) {
			console.error(e);
			rolesActionMessage.value = e.message || 'Failed to update group roles.';
			rolesActionMessageType.value = 'error';
		} finally {
			rolesSaving.value = false;
		}
	}

	async function submitAddRoles() {
		if (!addRoleIds.value.length) {
			addRolesError.value = 'Select at least one role.';
			return;
		}
		const nextIds = [...new Set([...selectedRoleIds.value, ...addRoleIds.value])];
		closeAddRolesDialog();
		const added = nextIds.length - selectedRoleIds.value.length;
		await persistGroupRoles(
			nextIds,
			added === 1 ? 'Added 1 role to the group.' : `Added ${added} roles to the group.`,
		);
	}

	async function revokeRole(role) {
		if (!canManageRoles.value || !confirm(`Revoke "${role.name}" from this group?`)) return;
		const nextIds = selectedRoleIds.value.filter((id) => id !== role.id);
		await persistGroupRoles(nextIds, `Revoked ${role.name} from the group.`);
	}

	function openMemberLookup() {
		memberLookup.value?.open();
	}

	async function persistMembers(nextIds, successMsg) {
		if (!canManage.value || !groupId.value) return;
		saving.value = true;
		actionMessage.value = '';
		actionMessageType.value = '';
		try {
			await waitForClient();
			const res = await window.client.setUserGroupMembers({
				groupId: groupId.value,
				userIds: nextIds,
			});
			if (res.standardResponse?.success) {
				actionMessage.value = successMsg || res.standardResponse.message || 'Members updated.';
				actionMessageType.value = 'success';
				await load();
			} else {
				actionMessage.value = res.standardResponse?.message || 'Failed to update members.';
				actionMessageType.value = 'error';
			}
		} catch (e) {
			console.error(e);
			actionMessage.value = e.message || 'Failed to update members.';
			actionMessageType.value = 'error';
		} finally {
			saving.value = false;
		}
	}

	async function onMembersPicked(picked) {
		const set = new Set(memberIds.value);
		for (const u of picked) {
			set.add(u.id);
		}
		const nextIds = Array.from(set);
		if (nextIds.length === memberIds.value.length) {
			actionMessage.value = 'No new users selected.';
			actionMessageType.value = 'note';
			return;
		}
		const added = nextIds.length - memberIds.value.length;
		await persistMembers(
			nextIds,
			added === 1 ? 'Added 1 user to the group.' : `Added ${added} users to the group.`
		);
	}

	async function removeMember(user) {
		if (!canManage.value || !confirm(`Remove "${user.username}" from this group?`)) return;
		const nextIds = memberIds.value.filter((id) => id !== user.id);
		await persistMembers(nextIds, `Removed ${user.username} from the group.`);
	}

	async function deleteGroup() {
		if (!canManage.value || !group.value) return;
		if (!confirm(`Delete group "${group.value.name}"? Members will be removed.`)) return;
		saving.value = true;
		errorMessage.value = '';
		try {
			await waitForClient();
			await window.client.deleteUserGroup({ groupId: group.value.id });
			router.push({ name: 'userGroups' });
		} catch (e) {
			console.error(e);
			errorMessage.value = e.message || 'Failed to delete group.';
			saving.value = false;
		}
	}

	watch(groupId, () => {
		load();
	}, { immediate: true });
</script>

<style scoped>
	.group-meta {
		display: grid;
		grid-template-columns: minmax(6rem, 10rem) 1fr;
		gap: 0.35rem 1rem;
		margin: 0 0 1.25rem;
		max-width: 28rem;
	}

	.group-meta dt {
		font-weight: 600;
		opacity: 0.85;
	}

	.group-meta dd {
		margin: 0;
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

	.members-table-wrap,
	.roles-table-wrap {
		margin-top: 0.5rem;
		margin-bottom: 1.5rem;
	}

	.role-tags {
		display: flex;
		flex-wrap: wrap;
		gap: 0.25rem;
		align-items: center;
	}

	.actions-cell {
		text-align: right;
	}

	.shared-accounts-table {
		width: 100%;
		max-width: 48rem;
	}

	.account-link {
		display: inline-flex;
		align-items: center;
		gap: 0.4rem;
	}

	.perm-col {
		width: 4.5rem;
		text-align: center;
	}

	.perm-yes {
		color: var(--pico-color-green-500, #2f9e44);
	}

	.perm-no {
		opacity: 0.35;
	}

	.username-link {
		font-weight: 600;
		text-decoration: none;
	}

	.username-link:hover {
		text-decoration: underline;
	}

	.muted {
		opacity: 0.8;
	}
</style>
