<template>
	<Section
		:title="pageTitle"
		subtitle="Assign RBAC roles and review effective permissions for this user"
		classes="user-details-roles"
	>
		<template #toolbar>
			<router-link :to="{ name: 'userDetails', params: { id: String(userId) } }" class="button neutral">
				<Icon icon="material-symbols:arrow-back" />
				Back to user
			</router-link>
			<router-link :to="{ name: 'settingsUsers' }" class="button neutral">
				<Icon icon="material-symbols:group" />
				All users
			</router-link>
			<button type="button" class="neutral" title="Refresh" :disabled="rolesLoading" @click="loadRoles">
				<Icon icon="material-symbols:refresh" />
			</button>
		</template>

		<div v-if="!userId" class="inline-notification error">Invalid user ID.</div>
		<div v-else-if="!canManageRoles" class="inline-notification error">
			You do not have permission to manage roles for this user.
		</div>
		<template v-else>
			<h3 class="subsection-title">Assign roles</h3>
			<p v-if="rolesLoading" class="keys-status">Loading roles…</p>
			<div v-else-if="rolesSaveError" class="inline-notification error">{{ rolesSaveError }}</div>
			<fieldset v-if="!rolesLoading && roles.length" class="role-checks">
				<legend>Roles for this user</legend>
				<label v-for="r in roles" :key="r.id" class="check-label">
					<input v-model="selectedRoleIds" type="checkbox" :value="r.id" />
					{{ r.name }}
					<span v-if="r.description" class="role-desc">— {{ r.description }}</span>
				</label>
				<button type="button" class="good" :disabled="rolesSaving" @click="saveUserRoles">Save roles</button>
			</fieldset>
			<p v-else-if="!rolesLoading" class="inline-notification note">No roles available.</p>

			<h3 class="subsection-title">Effective permissions</h3>
			<p v-if="rolesLoading" class="keys-status">Loading permissions…</p>
			<template v-else-if="permissions.length">
				<p v-if="userIsSuperuser" class="inline-notification note">
					This user has the <strong>superuser</strong> role — all permissions are granted.
				</p>
				<table class="perm-audit-table">
					<thead>
						<tr>
							<th class="perm-status-col">Status</th>
							<th>Permission</th>
							<th>Granted by</th>
						</tr>
					</thead>
					<tbody>
						<tr v-for="row in permissionAudit" :key="row.name">
							<td class="perm-status-col">
								<Icon v-if="row.granted" icon="material-symbols:check-circle" class="perm-granted" />
								<Icon v-else icon="material-symbols:cancel" class="perm-denied" />
							</td>
							<td><code>{{ row.name }}</code></td>
							<td>
								<span v-if="userIsSuperuser && row.grantingRoles.length === 0" class="role-tag superuser-tag">superuser</span>
								<template v-else>
									<span v-for="rn in row.grantingRoles" :key="rn" class="role-tag">{{ rn }}</span>
									<span v-if="row.grantingRoles.length === 0" class="muted">—</span>
								</template>
							</td>
						</tr>
					</tbody>
				</table>
			</template>
		</template>
	</Section>
</template>

<script setup>
	import { ref, computed, watch } from 'vue';
	import { useRoute } from 'vue-router';
	import { Icon } from '@iconify/vue';
	import Section from 'picocrank/vue/components/Section.vue';
	import { waitForClient } from '../javascript/util';

	const route = useRoute();
	const username = ref('');

	const statusPerms = ref([]);
	const statusSuper = ref(false);
	const roles = ref([]);
	const permissions = ref([]);
	const selectedRoleIds = ref([]);
	const rolesLoading = ref(false);
	const rolesSaving = ref(false);
	const rolesSaveError = ref('');

	const userId = computed(() => {
		const n = parseInt(String(route.params.id), 10);
		return Number.isFinite(n) && n > 0 ? n : 0;
	});

	const pageTitle = computed(() => (username.value ? `${username.value} — roles` : 'User roles'));

	const canManageRoles = computed(
		() => statusSuper.value || (Array.isArray(statusPerms.value) && statusPerms.value.includes('rbac.manage'))
	);

	const userIsSuperuser = computed(() => {
		return roles.value.some((r) => r.name === 'superuser' && selectedRoleIds.value.includes(r.id));
	});

	const permissionAudit = computed(() => {
		const roleById = {};
		for (const r of roles.value) {
			roleById[r.id] = r;
		}
		const userRoles = selectedRoleIds.value.map((id) => roleById[id]).filter(Boolean);

		return [...permissions.value]
			.sort((a, b) => a.name.localeCompare(b.name))
			.map((p) => {
				const grantingRoles = userRoles
					.filter((r) => r.permissionIds?.includes(p.id))
					.map((r) => r.name)
					.sort();
				const granted = userIsSuperuser.value || grantingRoles.length > 0;
				return { name: p.name, granted, grantingRoles };
			});
	});

	async function refreshStatus() {
		await waitForClient();
		const st = await window.client.getStatus({});
		statusPerms.value = st.rbacPermissions || [];
		statusSuper.value = Boolean(st.rbacIsSuperuser);
	}

	async function loadUserBasics() {
		if (!userId.value) return;
		try {
			await waitForClient();
			const res = await window.client.getUser({ userId: userId.value });
			username.value = res.user?.username || '';
		} catch {
			username.value = '';
		}
	}

	async function loadRoles() {
		if (!canManageRoles.value || !userId.value) return;
		rolesLoading.value = true;
		rolesSaveError.value = '';
		try {
			await waitForClient();
			const [rolesRes, userRolesRes, permsRes] = await Promise.all([
				window.client.listRbacRoles({}),
				window.client.getUserRbacRoles({ userId: userId.value }),
				window.client.listRbacPermissions({}),
			]);
			roles.value = rolesRes.roles || [];
			selectedRoleIds.value = [...(userRolesRes.roleIds || [])];
			permissions.value = permsRes.permissions || [];
		} catch (e) {
			rolesSaveError.value = e.message || 'Failed to load roles.';
		} finally {
			rolesLoading.value = false;
		}
	}

	async function saveUserRoles() {
		if (!canManageRoles.value) return;
		rolesSaving.value = true;
		rolesSaveError.value = '';
		try {
			await waitForClient();
			await window.client.setUserRbacRoles({
				userId: userId.value,
				roleIds: selectedRoleIds.value,
			});
			await loadRoles();
		} catch (e) {
			rolesSaveError.value = e.message || 'Failed to save roles.';
		} finally {
			rolesSaving.value = false;
		}
	}

	async function load() {
		await refreshStatus();
		await loadUserBasics();
		if (canManageRoles.value && userId.value) {
			await loadRoles();
		}
	}

	watch(userId, () => {
		load();
	}, { immediate: true });
</script>

<style scoped>
	.subsection-title {
		margin: 1.25rem 0 0.5rem;
		font-size: 1rem;
		font-weight: 600;
	}

	.subsection-title:first-of-type {
		margin-top: 0;
	}

	.keys-status {
		margin: 0 0 0.75rem;
		opacity: 0.85;
	}

	.role-checks {
		margin: 0.75rem 0;
		border: 1px solid var(--pico-muted-border-color, rgba(0, 0, 0, 0.12));
		padding: 0.75rem 1rem;
		border-radius: 4px;
		max-width: 40rem;
	}

	.check-label {
		display: block;
		margin: 0.25rem 0;
		font-size: 0.9em;
	}

	.role-desc {
		opacity: 0.7;
		font-size: 0.9em;
	}

	.perm-audit-table {
		width: 100%;
		max-width: 48rem;
		margin: 0.5rem 0 1.5rem;
	}

	.perm-status-col {
		width: 3.5rem;
		text-align: center;
	}

	.perm-granted {
		color: var(--pico-ins-color, #2a7d2e);
		font-size: 1.25em;
		vertical-align: middle;
	}

	.perm-denied {
		color: var(--pico-del-color, #9e2a2a);
		font-size: 1.25em;
		vertical-align: middle;
		opacity: 0.5;
	}

	.role-tag {
		display: inline-block;
		margin: 0.1rem 0.25rem 0.1rem 0;
		padding: 0.1rem 0.4rem;
		font-size: 0.8em;
		border-radius: 3px;
		background: var(--pico-muted-border-color, rgba(0, 0, 0, 0.08));
	}

	.superuser-tag {
		font-style: italic;
		opacity: 0.75;
	}

	.muted {
		opacity: 0.5;
	}
</style>
