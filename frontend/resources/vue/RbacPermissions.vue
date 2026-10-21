<template>
	<Section
		title="Permissions"
		subtitle="All RBAC permissions defined in the system. Permissions are granted to users through roles assigned to their groups."
		:icon="ShieldKeyIcon"
		classes="rbac-permissions settings-users"
		:padding="false"
	>

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
		</template>

		<NotificationBlock v-if="errorMessage" type="error" :message="errorMessage" />

		<Table
			class="permissions-table-wrap"
			:data="tableRows"
			:headers="tableHeaders"
			:loading="loading"
		>
			<template #empty>
				No permissions defined.
			</template>
			<template #cell-name="{ value }">
				<code>{{ value }}</code>
			</template>
			<template #cell-description="{ value }">
				{{ value || '—' }}
			</template>
			<template #cell-rolesLabel="{ row }">
				<span v-if="!row.roles.length" class="muted">—</span>
				<span v-else class="role-links">
					<template v-for="(role, index) in row.roles" :key="role.id">
						<router-link
							:to="{ name: 'rbacRoleDetails', params: { id: String(role.id) } }"
							class="role-link"
						>{{ role.name }}</router-link><span v-if="index < row.roles.length - 1">, </span>
					</template>
				</span>
			</template>
		</Table>
	</Section>
</template>

<script setup>
	import { ref, computed, onMounted } from 'vue';
	import { waitForClient } from '../javascript/util';
	import { HugeiconsIcon } from '@hugeicons/vue';
	import { RefreshIcon, ShieldKeyIcon } from '@hugeicons/core-free-icons';
	import Section from 'picocrank/vue/components/Section.vue';
	import NotificationBlock from 'picocrank/vue/components/NotificationBlock.vue';
	import Table from 'picocrank/vue/components/Table.vue';

	const iconStrokeWidth = 2.5;

	const permissions = ref([]);
	const roles = ref([]);
	const loading = ref(true);
	const errorMessage = ref('');
	const statusPerms = ref([]);
	const statusSuper = ref(false);

	const canView = computed(
		() => statusSuper.value || (Array.isArray(statusPerms.value) && statusPerms.value.includes('rbac.view'))
	);

	const tableHeaders = [
		{ key: 'name', label: 'Permission', sortable: true },
		{ key: 'description', label: 'Description', sortable: true },
		{ key: 'rolesLabel', label: 'Roles', sortable: true },
	];

	const tableRows = computed(() => {
		const rolesByPermissionId = new Map();
		for (const permission of permissions.value) {
			rolesByPermissionId.set(permission.id, []);
		}
		for (const role of roles.value) {
			for (const permissionId of role.permissionIds || []) {
				const list = rolesByPermissionId.get(permissionId);
				if (list) {
					list.push(role);
				}
			}
		}

		return [...permissions.value]
			.sort((a, b) => a.name.localeCompare(b.name))
			.map((permission) => {
				const permissionRoles = (rolesByPermissionId.get(permission.id) || [])
					.sort((a, b) => a.name.localeCompare(b.name));
				return {
					...permission,
					roles: permissionRoles,
					rolesLabel: permissionRoles.map((role) => role.name).join(', '),
				};
			});
	});

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
				errorMessage.value = 'You do not have permission to view RBAC permissions (rbac.view).';
				permissions.value = [];
				roles.value = [];
				return;
			}
			const [permissionsRes, rolesRes] = await Promise.all([
				window.client.listRbacPermissions({}),
				window.client.listRbacRoles({}),
			]);
			permissions.value = permissionsRes.permissions || [];
			roles.value = rolesRes.roles || [];
		} catch (e) {
			console.error(e);
			errorMessage.value = e.message || 'Failed to load permissions.';
		} finally {
			loading.value = false;
		}
	}

	onMounted(() => {
		loadAll();
	});
</script>

<style scoped>


	.permissions-table-wrap {
		margin-top: 0.5rem;
		margin-bottom: 1.5rem;
	}

	.role-links {
		display: inline;
	}

	.role-link {
		font-weight: 600;
		text-decoration: none;
	}

	.role-link:hover {
		text-decoration: underline;
	}

	.muted {
		opacity: 0.8;
	}
</style>
