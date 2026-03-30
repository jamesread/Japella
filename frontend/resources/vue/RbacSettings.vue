<template>
	<Section
		title="Roles & permissions"
		subtitle="Manage RBAC roles, link them to permissions, and assign roles to users. Superuser has all permissions."
		classes="rbac-settings"
	>
		<template #toolbar>
			<button type="button" class="neutral" title="Refresh" :disabled="loading" @click="loadAll">
				<Icon icon="material-symbols:refresh" />
			</button>
		</template>

		<div v-if="errorMessage" class="inline-notification error">{{ errorMessage }}</div>

		<div v-if="loading && !permissions.length" class="muted">Loading…</div>

		<template v-else>
			<h3 class="subsection-title">Permissions</h3>
			<ul class="perm-list">
				<li v-for="p in permissions" :key="p.id">
					<code>{{ p.name }}</code>
					<span class="perm-desc">{{ p.description }}</span>
				</li>
			</ul>

			<h3 class="subsection-title">Roles</h3>
			<p v-if="!canManage" class="inline-notification note">You can view roles. Editing requires the rbac.manage permission.</p>

			<table v-if="roles.length" class="roles-table">
				<thead>
					<tr>
						<th>Name</th>
						<th>Description</th>
						<th>Permissions</th>
						<th v-if="canManage" class="small">Actions</th>
					</tr>
				</thead>
				<tbody>
					<tr v-for="r in roles" :key="r.id">
						<td>
							<strong>{{ r.name }}</strong>
							<span v-if="r.name === 'superuser'" class="badge">all access</span>
						</td>
						<td>{{ r.description }}</td>
						<td class="perm-cells">{{ permissionLabels(r.permissionIds) }}</td>
						<td v-if="canManage" align="right">
							<button
								v-if="r.name !== 'superuser' && r.name !== 'member'"
								type="button"
								class="neutral small"
								@click="openEditRole(r)"
							>
								Edit
							</button>
							<button
								v-if="r.name !== 'superuser' && r.name !== 'member'"
								type="button"
								class="bad small"
								@click="deleteRole(r)"
							>
								Delete
							</button>
						</td>
					</tr>
				</tbody>
			</table>

			<div v-if="canManage" class="create-role-panel">
				<h4 class="subsection-title">Create role</h4>
				<div class="form-row">
					<label for="new-role-name">Name</label>
					<input id="new-role-name" v-model="newRole.name" type="text" autocomplete="off" />
				</div>
				<div class="form-row">
					<label for="new-role-desc">Description</label>
					<input id="new-role-desc" v-model="newRole.description" type="text" />
				</div>
				<fieldset class="perm-checks">
					<legend>Permissions</legend>
					<label v-for="p in permissions" :key="'n-' + p.id" class="check-label">
						<input v-model="newRole.permissionIds" type="checkbox" :value="p.id" />
						{{ p.name }}
					</label>
				</fieldset>
				<button type="button" class="good" :disabled="saving || !newRole.name.trim()" @click="createRole">
					Create role
				</button>
			</div>

			<h3 class="subsection-title">User role assignment</h3>
			<div class="user-roles-row">
				<label for="ur-user">User</label>
				<select id="ur-user" v-model.number="userRolesUserId" @change="loadUserRoles">
					<option :value="0">Select user…</option>
					<option v-for="u in users" :key="u.id" :value="u.id">{{ u.username }} ({{ u.id }})</option>
				</select>
			</div>
			<fieldset v-if="userRolesUserId && canManage" class="perm-checks">
				<legend>Roles for selected user</legend>
				<label v-for="r in roles" :key="'ur-' + r.id" class="check-label">
					<input v-model="selectedUserRoleIds" type="checkbox" :value="r.id" />
					{{ r.name }}
				</label>
				<button type="button" class="good" :disabled="saving" @click="saveUserRoles">Save user roles</button>
			</fieldset>
			<p v-else-if="userRolesUserId && !canManage" class="inline-notification note">You need rbac.manage to change user roles.</p>
		</template>
	</Section>

	<dialog ref="editDialog" class="dialog" @close="editRole = null">
		<form v-if="editRole" @submit.prevent="saveEditedRole">
			<h3>Edit role: {{ editRole.name }}</h3>
			<label>Name</label>
			<input v-model="editRole.name" type="text" required :disabled="editRole.originalName === 'member'" />
			<label>Description</label>
			<input v-model="editRole.description" type="text" />
			<fieldset class="perm-checks">
				<legend>Permissions</legend>
				<label v-for="p in permissions" :key="'e-' + p.id" class="check-label">
					<input v-model="editRole.permissionIds" type="checkbox" :value="p.id" />
					{{ p.name }}
				</label>
			</fieldset>
			<div class="dialog-actions">
				<button type="button" class="neutral" @click="editDialog?.close()">Cancel</button>
				<button type="submit" class="good" :disabled="saving">Save</button>
			</div>
		</form>
	</dialog>
</template>

<script setup>
	import { ref, computed, onMounted } from 'vue';
	import { waitForClient } from '../javascript/util';
	import { Icon } from '@iconify/vue';
	import Section from 'picocrank/vue/components/Section.vue';

	const permissions = ref([]);
	const roles = ref([]);
	const users = ref([]);
	const loading = ref(true);
	const saving = ref(false);
	const errorMessage = ref('');
	const statusPerms = ref([]);
	const statusSuper = ref(false);

	const newRole = ref({ name: '', description: '', permissionIds: [] });
	const editRole = ref(null);
	const editDialog = ref(null);
	const userRolesUserId = ref(0);
	const selectedUserRoleIds = ref([]);

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

	function permissionLabels(ids) {
		if (!ids?.length) {
			return '—';
		}
		return ids
			.map((id) => permById.value[id]?.name || id)
			.sort()
			.join(', ');
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
			const [pr, rr, ur] = await Promise.all([
				window.client.listRbacPermissions({}),
				window.client.listRbacRoles({}),
				window.client.getUsers({})
			]);
			permissions.value = pr.permissions || [];
			roles.value = rr.roles || [];
			users.value = ur.users || [];
		} catch (e) {
			console.error(e);
			errorMessage.value = e.message || 'Failed to load RBAC data.';
		} finally {
			loading.value = false;
		}
	}

	async function loadUserRoles() {
		if (!userRolesUserId.value || !canManage.value) {
			selectedUserRoleIds.value = [];
			return;
		}
		try {
			await waitForClient();
			const res = await window.client.getUserRbacRoles({ userId: userRolesUserId.value });
			selectedUserRoleIds.value = [...(res.roleIds || [])];
		} catch (e) {
			console.error(e);
			errorMessage.value = e.message || 'Failed to load user roles.';
		}
	}

	async function saveUserRoles() {
		if (!canManage.value) {
			return;
		}
		saving.value = true;
		errorMessage.value = '';
		try {
			await waitForClient();
			await window.client.setUserRbacRoles({
				userId: userRolesUserId.value,
				roleIds: selectedUserRoleIds.value
			});
			await loadAll();
		} catch (e) {
			console.error(e);
			errorMessage.value = e.message || 'Failed to save user roles.';
		} finally {
			saving.value = false;
		}
	}

	async function createRole() {
		if (!canManage.value) {
			return;
		}
		saving.value = true;
		errorMessage.value = '';
		try {
			await waitForClient();
			await window.client.createRbacRole({
				name: newRole.value.name.trim(),
				description: newRole.value.description.trim(),
				permissionIds: newRole.value.permissionIds
			});
			newRole.value = { name: '', description: '', permissionIds: [] };
			await loadAll();
		} catch (e) {
			console.error(e);
			errorMessage.value = e.message || 'Failed to create role.';
		} finally {
			saving.value = false;
		}
	}

	function openEditRole(r) {
		editRole.value = {
			id: r.id,
			name: r.name,
			originalName: r.name,
			description: r.description || '',
			permissionIds: [...(r.permissionIds || [])]
		};
		editDialog.value?.showModal();
	}

	async function saveEditedRole() {
		if (!editRole.value || !canManage.value) {
			return;
		}
		saving.value = true;
		errorMessage.value = '';
		try {
			await waitForClient();
			await window.client.updateRbacRole({
				roleId: editRole.value.id,
				name: editRole.value.name.trim(),
				description: editRole.value.description.trim(),
				permissionIds: editRole.value.permissionIds
			});
			editDialog.value?.close();
			await loadAll();
		} catch (e) {
			console.error(e);
			errorMessage.value = e.message || 'Failed to update role.';
		} finally {
			saving.value = false;
		}
	}

	async function deleteRole(r) {
		if (!canManage.value || !confirm(`Delete role "${r.name}"? Users lose this role assignment.`)) {
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
	.subsection-title {
		margin: 1.25rem 0 0.5rem;
		font-size: 1.05rem;
		font-weight: 600;
	}

	.perm-list {
		margin: 0 0 1rem;
		padding-left: 1.2rem;
		max-width: 48rem;
	}

	.perm-list li {
		margin-bottom: 0.35rem;
	}

	.perm-desc {
		margin-left: 0.5rem;
		opacity: 0.85;
		font-size: 0.9em;
	}

	.roles-table {
		width: 100%;
		max-width: 56rem;
		margin-bottom: 1.5rem;
	}

	.perm-cells {
		font-size: 0.85em;
		max-width: 24rem;
	}

	.badge {
		margin-left: 0.35rem;
		font-size: 0.75rem;
		font-weight: 600;
		opacity: 0.75;
	}

	.create-role-panel {
		margin-bottom: 2rem;
		padding: 1rem 0;
		border-top: 1px solid var(--pico-muted-border-color, rgba(0, 0, 0, 0.12));
		max-width: 40rem;
	}

	.form-row {
		display: grid;
		grid-template-columns: 8rem 1fr;
		gap: 0.5rem 1rem;
		align-items: center;
		margin-bottom: 0.5rem;
	}

	.perm-checks {
		margin: 0.75rem 0;
		border: 1px solid var(--pico-muted-border-color, rgba(0, 0, 0, 0.12));
		padding: 0.75rem 1rem;
		border-radius: 4px;
	}

	.check-label {
		display: block;
		margin: 0.25rem 0;
		font-size: 0.9em;
	}

	.user-roles-row {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		gap: 0.75rem;
		margin-bottom: 0.75rem;
	}

	.user-roles-row select {
		min-width: 14rem;
	}

	.dialog-actions {
		display: flex;
		gap: 0.75rem;
		margin-top: 1rem;
	}

	.muted {
		opacity: 0.8;
	}
</style>
