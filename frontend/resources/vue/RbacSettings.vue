<template>
	<CreateRbacRole ref="createRoleDialog" :permissions="permissions" @created="onRoleCreated" />

	<Section
		subtitle="Manage RBAC roles, link them to permissions, and assign roles to user groups. Superuser has all permissions."
		classes="rbac-settings settings-users"
		:padding="false"
	>
		<template #title>
			<span class="section-title-with-icon">
				<HugeiconsIcon :icon="WebSecurityIcon" width="22" height="22" aria-hidden="true" />
				Roles &amp; Permissions
			</span>
		</template>

		<template #toolbar>
			<button
				type="button"
				class="inline-icon neutral"
				aria-label="Refresh"
				title="Refresh"
				:disabled="loading"
				@click="loadAll"
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
				aria-label="Create role"
				title="Create role"
				:disabled="loading"
				@click="openCreateRoleDialog"
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

		<div v-if="errorMessage" class="inline-notification error list-banner-pad">{{ errorMessage }}</div>
		<div v-if="loading && !roles.length" class="list-banner-pad muted">Loading…</div>

		<template v-else>
			<p v-if="!canManage" class="inline-notification note list-banner-pad">
				You can view roles. Editing requires the rbac.manage permission.
			</p>

			<p v-if="!roles.length" class="inline-notification note list-banner-pad">No roles yet.</p>

			<Table
				v-else
				class="roles-table-wrap"
				row-clickable
				:data="tableRows"
				:headers="tableHeaders"
				@row-click="onRoleRowClick"
			>
				<template #cell-name="{ row, value }">
					<router-link :to="{ name: 'rbacRoleDetails', params: { id: String(row.id) } }" class="username-link">
						<strong>{{ value }}</strong>
					</router-link>
				</template>
				<template #cell-permissionsLabel="{ value }">
					<span class="perm-cells">{{ value }}</span>
				</template>
				<template #cell-tags="{ row }">
					<span class="role-tags">
						<span v-if="isSystemRole(row.name)" class="tag fg-note">system role</span>
						<span v-if="row.name === 'superuser'" class="tag fg-good">all access</span>
						<span v-if="!isSystemRole(row.name)" class="muted">—</span>
					</span>
				</template>
				<template #cell-userCount="{ row }">
					<span class="used-by-cell">{{ row.usedByLabel }}</span>
				</template>
				<template #cell-actions="{ row }">
					<div v-if="canDeleteRole(row)" class="actions-cell">
						<button type="button" class="bad small" :disabled="saving" @click="deleteRole(row)">
							Delete
						</button>
					</div>
				</template>
			</Table>
		</template>
	</Section>
</template>

<script setup>
	import { ref, computed, onMounted } from 'vue';
	import { useRouter } from 'vue-router';
	import { waitForClient } from '../javascript/util';
	import { HugeiconsIcon } from '@hugeicons/vue';
	import { Add01Icon, RefreshIcon, WebSecurityIcon } from '@hugeicons/core-free-icons';
	import Section from 'picocrank/vue/components/Section.vue';
	import Table from './picocrank/TableWithRowClick.vue';
	import CreateRbacRole from './CreateRbacRole.vue';

	const iconStrokeWidth = 2.5;
	const router = useRouter();

	const permissions = ref([]);
	const roles = ref([]);
	const loading = ref(true);
	const saving = ref(false);
	const errorMessage = ref('');
	const statusPerms = ref([]);
	const statusSuper = ref(false);

	const createRoleDialog = ref(null);

	const canView = computed(
		() => statusSuper.value || (Array.isArray(statusPerms.value) && statusPerms.value.includes('rbac.view'))
	);
	const canManage = computed(
		() => statusSuper.value || (Array.isArray(statusPerms.value) && statusPerms.value.includes('rbac.manage'))
	);

	const permById = computed(() => {
		const m = {};
		for (const p of permissions.value) {
			m[p.id] = p;
		}
		return m;
	});

	const tableHeaders = computed(() => {
		const headers = [
			{ key: 'name', label: 'Name', sortable: true },
			{ key: 'description', label: 'Description', sortable: true },
			{ key: 'permissionsLabel', label: 'Permissions', sortable: true },
			{ key: 'tags', label: 'Tags', sortable: false, width: '11rem' },
			{ key: 'userCount', label: 'Used By', sortable: true, width: '11rem' },
		];
		if (canManage.value) {
			headers.push({ key: 'actions', label: 'Actions', sortable: false, width: '6rem' });
		}
		return headers;
	});

	function formatUsedBy(groupCount, userCount) {
		const g = Number(groupCount) || 0;
		const u = Number(userCount) || 0;
		const groupLabel = g === 1 ? 'group' : 'groups';
		const userLabel = u === 1 ? 'user' : 'users';
		return `${g} ${groupLabel} (${u} ${userLabel})`;
	}

	const tableRows = computed(() =>
		roles.value.map((r) => ({
			...r,
			description: r.description || '—',
			permissionsLabel: permissionLabels(r.permissionIds),
			groupCount: r.groupCount ?? 0,
			userCount: r.userCount ?? 0,
			usedByLabel: formatUsedBy(r.groupCount, r.userCount),
		})),
	);

	function canDeleteRole(r) {
		return canManage.value && r.name !== 'superuser' && r.name !== 'member';
	}

	function isSystemRole(name) {
		return name === 'superuser' || name === 'member';
	}

	function permissionLabels(ids) {
		if (!ids?.length) {
			return '—';
		}
		return ids
			.map((id) => permById.value[id]?.name || id)
			.sort()
			.join(', ');
	}

	function onRoleRowClick(r) {
		router.push({ name: 'rbacRoleDetails', params: { id: String(r.id) } });
	}

	async function refreshStatus() {
		await waitForClient();
		const st = await window.client.getStatus({});
		statusPerms.value = st.rbacPermissions || [];
		statusSuper.value = Boolean(st.rbacIsSuperuser);
	}

	async function loadAll() {
		errorMessage.value = '';
		loading.value = true;
		try {
			await waitForClient();
			await refreshStatus();
			if (!canView.value) {
				errorMessage.value = 'You do not have permission to view RBAC (rbac.view).';
				return;
			}
			const [pr, rr] = await Promise.all([
				window.client.listRbacPermissions({}),
				window.client.listRbacRoles({}),
			]);
			permissions.value = pr.permissions || [];
			roles.value = rr.roles || [];
		} catch (e) {
			console.error(e);
			errorMessage.value = e.message || 'Failed to load RBAC data.';
		} finally {
			loading.value = false;
		}
	}

	function openCreateRoleDialog() {
		createRoleDialog.value?.open();
	}

	function onRoleCreated(roleId) {
		if (roleId) {
			router.push({ name: 'rbacRoleDetails', params: { id: String(roleId) } });
			return;
		}
		loadAll();
	}

	async function deleteRole(r) {
		if (!canManage.value || !confirm(`Delete role "${r.name}"? Groups lose this role assignment.`)) {
			return;
		}
		saving.value = true;
		errorMessage.value = '';
		try {
			await waitForClient();
			await window.client.deleteRbacRole({ roleId: r.id });
			await loadAll();
		} catch (e) {
			console.error(e);
			errorMessage.value = e.message || 'Failed to delete role.';
		} finally {
			saving.value = false;
		}
	}

	onMounted(() => {
		loadAll();
	});
</script>

<style scoped>
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

	.roles-table-wrap {
		margin-top: 0.5rem;
		margin-bottom: 1.5rem;
	}

	.actions-cell {
		text-align: right;
	}

	.perm-cells {
		font-size: 0.85em;
	}

	.role-tags {
		display: flex;
		flex-wrap: wrap;
		gap: 0.25rem;
		align-items: center;
	}

	.used-by-cell {
		font-size: 0.9em;
		white-space: nowrap;
	}

	.muted {
		opacity: 0.8;
	}
</style>
