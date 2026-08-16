<template>
	<Section
		:title="role?.name || 'Role'"
		subtitle="View and edit permissions granted by this role."
		classes="role-details"
	>
		<template #toolbar>
			<router-link :to="{ name: 'rbacSettings' }" class="button inline-icon neutral">
				<HugeiconsIcon
					:icon="ArrowLeft01Icon"
					width="1em"
					height="1em"
					:strokeWidth="iconStrokeWidth"
					aria-hidden="true"
				/>
				<span>Back to Roles &amp; Permissions</span>
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
				v-if="canDelete"
				type="button"
				class="inline-icon bad"
				:disabled="saving || loading"
				@click="deleteRole"
			>
				<HugeiconsIcon
					:icon="Delete02Icon"
					width="1em"
					height="1em"
					:strokeWidth="iconStrokeWidth"
					aria-hidden="true"
				/>
				<span>Delete role</span>
			</button>
		</template>

		<div v-if="errorMessage" class="inline-notification error">{{ errorMessage }}</div>
		<div v-if="actionMessage" class="inline-notification" :class="actionMessageType">{{ actionMessage }}</div>

		<div v-if="loading" class="muted">Loading…</div>
		<div v-else-if="!role" class="inline-notification note">Role not found.</div>
		<template v-else>
			<dl class="role-meta">
				<dt>Role ID</dt>
				<dd>{{ role.id }}</dd>
				<dt>Name</dt>
				<dd>
					{{ role.name }}
					<span v-if="role.name === 'superuser'" class="badge">all access</span>
					<span v-else-if="isSystemRole" class="badge">system role</span>
				</dd>
				<dt>Description</dt>
				<dd>{{ role.description || '—' }}</dd>
				<dt>Groups</dt>
				<dd>{{ assignedGroups.length }}</dd>
				<dt>Users via groups</dt>
				<dd>{{ assignedUsers.length }}</dd>
			</dl>

			<h3 class="subsection-title">Permissions</h3>
			<p v-if="role.name === 'superuser'" class="inline-notification note">
				The <strong>superuser</strong> role grants every permission in the system. Its permission set cannot be edited.
			</p>
			<p v-else-if="!canManage" class="inline-notification note">
				You can view permissions. Editing requires the rbac.manage permission.
			</p>

			<FormLayout v-if="canEdit" @submit.prevent="saveRole">
				<FormField label="Name" for="role-name" :disabled="saving || !canEditName">
					<input
						id="role-name"
						v-model="form.name"
						type="text"
						required
						:disabled="!canEditName || saving"
					/>
				</FormField>
				<FormField label="Description" for="role-desc" :disabled="saving">
					<input id="role-desc" v-model="form.description" type="text" :disabled="saving" />
				</FormField>
				<FormField label="Granted permissions" fake :disabled="saving">
					<table class="perm-table data-table">
						<thead>
							<tr>
								<th class="perm-col-check" scope="col"><span class="a11yhidden">Grant</span></th>
								<th scope="col">Permission</th>
								<th scope="col">Description</th>
							</tr>
						</thead>
						<tbody>
							<tr v-for="p in permissionsSorted" :key="p.id">
								<td class="perm-col-check">
									<input
										v-model="form.permissionIds"
										type="checkbox"
										:value="p.id"
										:disabled="saving"
										:aria-label="'Grant ' + p.name"
									/>
								</td>
								<td><code>{{ p.name }}</code></td>
								<td>{{ p.description || '—' }}</td>
							</tr>
						</tbody>
					</table>
				</FormField>
				<template #actions>
					<button type="submit" class="good" :disabled="saving || !form.name.trim()">
						{{ saving ? 'Saving…' : 'Save role' }}
					</button>
				</template>
			</FormLayout>

			<table v-else class="perm-table data-table perm-readonly">
				<thead>
					<tr>
						<th class="perm-status-col">Granted</th>
						<th>Permission</th>
						<th>Description</th>
					</tr>
				</thead>
				<tbody>
					<tr v-for="p in permissionsSorted" :key="p.id">
						<td class="perm-status-col">
							<Icon
								v-if="role.name === 'superuser' || form.permissionIds.includes(p.id)"
								icon="material-symbols:check"
								class="perm-yes"
							/>
							<Icon v-else icon="material-symbols:close" class="perm-no" />
						</td>
						<td><code>{{ p.name }}</code></td>
						<td>{{ p.description || '—' }}</td>
					</tr>
				</tbody>
			</table>
		</template>
	</Section>

	<Section
		v-if="!loading && role"
		subtitle="User groups that assign this role. Add groups here or manage assignments on group detail pages."
		classes="role-details-groups settings-users"
		:padding="assignedGroups.length === 0"
	>
		<template #title>
			<span class="section-title-with-icon">
				<HugeiconsIcon :icon="UserGroupIcon" width="22" height="22" aria-hidden="true" />
				Groups with this role
			</span>
		</template>

		<template #toolbar>
			<button
				type="button"
				class="inline-icon neutral"
				aria-label="Refresh"
				title="Refresh"
				:disabled="loading || groupsSaving"
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
				aria-label="Add group"
				title="Add group"
				:disabled="loading || groupsSaving"
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

		<div
			v-if="groupsActionMessage"
			class="inline-notification list-banner-pad"
			:class="groupsActionMessageType"
		>{{ groupsActionMessage }}</div>

		<p v-if="!assignedGroups.length" class="inline-notification note">No groups are assigned this role.</p>

		<Table
			v-else
			class="groups-table-wrap"
			row-clickable
			:data="assignedGroups"
			:headers="groupTableHeaders"
			@row-click="openGroupDetails"
		>
			<template #cell-name="{ value }">
				<strong>{{ value }}</strong>
			</template>
		</Table>
	</Section>

	<UserGroupLookupDialog
		ref="groupLookup"
		title="Assign groups to this role"
		:subtitle="role ? `Select groups to grant the “${role.name}” role.` : ''"
		multiple
		confirm-label="Add groups"
		:exclude-group-ids="assignedGroupIds"
		@picked="onRoleGroupsPicked"
	/>

	<Section
		v-if="!loading && role"
		subtitle="Users who inherit this role through group membership."
		classes="role-details-users settings-users"
		:padding="assignedUsers.length === 0"
	>
		<template #title>
			<span class="section-title-with-icon">
				<HugeiconsIcon :icon="UserMultiple02Icon" width="22" height="22" aria-hidden="true" />
				Users via groups
			</span>
		</template>

		<template #toolbar>
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
		</template>

		<p v-if="!assignedUsers.length" class="inline-notification note">No users inherit this role via groups.</p>

		<Table
			v-else
			class="users-table-wrap"
			row-clickable
			:data="assignedUsers"
			:headers="userTableHeaders"
			@row-click="openUserDetails"
		>
			<template #cell-username="{ value }">
				<strong>{{ value }}</strong>
			</template>
		</Table>
	</Section>
</template>

<script setup>
	import { ref, computed, watch, reactive } from 'vue';
	import { useRoute, useRouter } from 'vue-router';
	import { Icon } from '@iconify/vue';
	import { HugeiconsIcon } from '@hugeicons/vue';
	import { ArrowLeft01Icon, Add01Icon, Delete02Icon, RefreshIcon, UserGroupIcon, UserMultiple02Icon } from '@hugeicons/core-free-icons';
	import Section from 'picocrank/vue/components/Section.vue';
	import FormLayout from 'picocrank/vue/components/FormLayout.vue';
	import FormField from 'picocrank/vue/components/FormField.vue';
	import Table from './picocrank/TableWithRowClick.vue';
	import UserGroupLookupDialog from './UserGroupLookupDialog.vue';
	import { waitForClient } from '../javascript/util';

	const iconStrokeWidth = 2.5;
	const route = useRoute();
	const router = useRouter();

	const role = ref(null);
	const permissions = ref([]);
	const assignedGroups = ref([]);
	const assignedUsers = ref([]);
	const loading = ref(true);
	const saving = ref(false);
	const groupsSaving = ref(false);
	const errorMessage = ref('');
	const actionMessage = ref('');
	const actionMessageType = ref('');
	const groupsActionMessage = ref('');
	const groupsActionMessageType = ref('');
	const statusPerms = ref([]);
	const statusSuper = ref(false);
	const groupLookup = ref(null);

	const form = reactive({
		name: '',
		description: '',
		permissionIds: [],
	});

	const roleId = computed(() => {
		const n = parseInt(String(route.params.id), 10);
		return Number.isFinite(n) && n > 0 ? n : 0;
	});

	const canView = computed(
		() => statusSuper.value || (Array.isArray(statusPerms.value) && statusPerms.value.includes('rbac.view'))
	);
	const canManage = computed(
		() => statusSuper.value || (Array.isArray(statusPerms.value) && statusPerms.value.includes('rbac.manage'))
	);

	const isSystemRole = computed(() => role.value?.name === 'superuser' || role.value?.name === 'member');
	const canEditName = computed(() => canManage.value && role.value?.name !== 'superuser' && role.value?.name !== 'member');
	const canEdit = computed(() => canManage.value && role.value?.name !== 'superuser');
	const canDelete = computed(() => canManage.value && role.value?.name !== 'superuser' && role.value?.name !== 'member');

	const permissionsSorted = computed(() =>
		[...permissions.value].sort((a, b) =>
			String(a.name || '').localeCompare(String(b.name || ''), undefined, { sensitivity: 'base' }),
		),
	);

	const groupTableHeaders = [
		{ key: 'name', label: 'Group', sortable: true },
		{ key: 'memberCount', label: 'Members', sortable: true, width: '8rem' },
	];

	const userTableHeaders = [
		{ key: 'username', label: 'Username', sortable: true },
		{ key: 'id', label: 'User ID', sortable: true, width: '8rem' },
	];

	const assignedGroupIds = computed(() => assignedGroups.value.map((g) => g.id));

	function openGroupLookup() {
		groupLookup.value?.open();
	}

	async function onRoleGroupsPicked(groups) {
		if (!canManage.value || !role.value || !groups.length) {
			return;
		}

		groupsSaving.value = true;
		groupsActionMessage.value = '';
		groupsActionMessageType.value = '';

		try {
			await waitForClient();
			for (const group of groups) {
				const res = await window.client.getUserGroupRbacRoles({ groupId: group.id });
				const roleIds = [...(res.roleIds || [])];
				if (roleIds.includes(role.value.id)) {
					continue;
				}
				await window.client.setUserGroupRbacRoles({
					groupId: group.id,
					roleIds: [...roleIds, role.value.id],
				});
			}
			groupsActionMessage.value = groups.length === 1
				? `Added role to ${groups[0].name}.`
				: `Added role to ${groups.length} groups.`;
			groupsActionMessageType.value = 'success';
			await load();
		} catch (e) {
			console.error(e);
			groupsActionMessage.value = e.message || 'Failed to assign role to groups.';
			groupsActionMessageType.value = 'error';
		} finally {
			groupsSaving.value = false;
		}
	}

	function openGroupDetails(row) {
		router.push({ name: 'userGroupDetails', params: { id: String(row.id) } });
	}

	function openUserDetails(row) {
		router.push({ name: 'userDetails', params: { id: String(row.id) } });
	}

	function resetFormFromRole(r) {
		form.name = r.name || '';
		form.description = r.description || '';
		form.permissionIds = [...(r.permissionIds || [])];
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
		groupsActionMessage.value = '';
		groupsActionMessageType.value = '';
		loading.value = true;
		role.value = null;
		assignedGroups.value = [];
		assignedUsers.value = [];

		if (!roleId.value) {
			errorMessage.value = 'Invalid role ID.';
			loading.value = false;
			return;
		}

		try {
			await waitForClient();
			await refreshStatus();
			if (!canView.value) {
				errorMessage.value = 'You do not have permission to view RBAC (rbac.view).';
				return;
			}

			const [rolesRes, permsRes, usersRes, roleUsersRes, roleGroupsRes, groupsRes] = await Promise.all([
				window.client.listRbacRoles({}),
				window.client.listRbacPermissions({}),
				window.client.getUsers({}),
				window.client.getRbacRoleUsers({ roleId: roleId.value }),
				window.client.getRbacRoleGroups({ roleId: roleId.value }),
				window.client.listUserGroups({}),
			]);

			const found = (rolesRes.roles || []).find((r) => r.id === roleId.value) || null;
			role.value = found;
			permissions.value = permsRes.permissions || [];

			if (found) {
				resetFormFromRole(found);
			}

			const userMap = new Map();
			for (const u of usersRes.users || []) {
				userMap.set(u.id, u);
			}

			const ids = roleUsersRes.userIds || [];
			assignedUsers.value = ids
				.map((id) => userMap.get(id) || { id, username: `User #${id}` })
				.sort((a, b) => a.username.localeCompare(b.username));

			const groupMap = new Map();
			for (const g of groupsRes.groups || []) {
				groupMap.set(g.id, g);
			}
			assignedGroups.value = (roleGroupsRes.groupIds || [])
				.map((id) => groupMap.get(id) || { id, name: `Group #${id}`, memberCount: 0 })
				.sort((a, b) => a.name.localeCompare(b.name));
		} catch (e) {
			console.error(e);
			errorMessage.value = e.message || 'Failed to load role.';
		} finally {
			loading.value = false;
		}
	}

	async function saveRole() {
		if (!canEdit.value || !role.value) {
			return;
		}
		saving.value = true;
		actionMessage.value = '';
		actionMessageType.value = '';
		try {
			await waitForClient();
			await window.client.updateRbacRole({
				roleId: role.value.id,
				name: form.name.trim(),
				description: form.description.trim(),
				permissionIds: form.permissionIds,
			});
			actionMessage.value = 'Role updated.';
			actionMessageType.value = 'success';
			await load();
		} catch (e) {
			console.error(e);
			actionMessage.value = e.message || 'Failed to update role.';
			actionMessageType.value = 'error';
		} finally {
			saving.value = false;
		}
	}

	async function deleteRole() {
		if (!canDelete.value || !role.value) {
			return;
		}
		if (!confirm(`Delete role "${role.value.name}"? Groups lose this role assignment.`)) {
			return;
		}
		saving.value = true;
		errorMessage.value = '';
		try {
			await waitForClient();
			await window.client.deleteRbacRole({ roleId: role.value.id });
			router.push({ name: 'rbacSettings' });
		} catch (e) {
			console.error(e);
			errorMessage.value = e.message || 'Failed to delete role.';
		} finally {
			saving.value = false;
		}
	}

	watch(roleId, () => {
		load();
	}, { immediate: true });
</script>

<style scoped>
	.role-meta {
		display: grid;
		grid-template-columns: auto 1fr;
		gap: 0.35rem 1.25rem;
		max-width: 40rem;
		margin: 0 0 1.25rem;
	}

	.role-meta dt {
		font-weight: 600;
		margin: 0;
	}

	.role-meta dd {
		margin: 0;
	}

	.badge {
		margin-left: 0.35rem;
		font-size: 0.75rem;
		font-weight: 600;
		opacity: 0.75;
	}

	.subsection-title {
		margin: 1.25rem 0 0.5rem;
		font-size: 1.05rem;
		font-weight: 600;
	}

	.perm-table {
		width: 100%;
		margin: 0.25rem 0 1rem;
	}

	.perm-col-check {
		width: 2.5rem;
		vertical-align: middle;
		text-align: center;
	}

	.perm-table td.perm-col-check input {
		margin: 0;
	}

	.perm-status-col {
		width: 4rem;
		text-align: center;
	}

	.perm-yes {
		color: var(--pico-ins-color, #2a7d2e);
	}

	.perm-no {
		color: var(--pico-del-color, #9e2a2a);
		opacity: 0.5;
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

	.groups-table-wrap,
	.users-table-wrap {
		margin-top: 0.5rem;
		margin-bottom: 1.5rem;
	}

	.muted {
		opacity: 0.8;
	}
</style>
