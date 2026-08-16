<template>
	<Section
		:title="pageTitle"
		subtitle="Assign groups to control this user's access. Roles and permissions are inherited from group membership."
		classes="user-details-roles"
	>
		<template #toolbar>
			<router-link :to="{ name: 'userDetails', params: { id: String(userId) } }" class="button inline-icon neutral">
				<HugeiconsIcon
					:icon="ArrowLeft01Icon"
					width="1em"
					height="1em"
					:strokeWidth="iconStrokeWidth"
					aria-hidden="true"
				/>
				<span>Back to user</span>
			</router-link>
			<router-link :to="{ name: 'settingsUsers' }" class="button inline-icon neutral">
				<HugeiconsIcon
					:icon="UserGroupIcon"
					width="1em"
					height="1em"
					:strokeWidth="iconStrokeWidth"
					aria-hidden="true"
				/>
				<span>All users</span>
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
		</template>

		<div v-if="!userId" class="inline-notification error">Invalid user ID.</div>
		<div v-else-if="errorMessage" class="inline-notification error">{{ errorMessage }}</div>
		<div v-else-if="loading" class="muted">Loading…</div>
		<template v-else>
			<h3 class="subsection-title">Group membership</h3>
			<p v-if="!canManageGroups" class="inline-notification note">
				You can view group membership. Editing requires the usergroups.manage permission.
			</p>
			<p v-else-if="!allGroups.length" class="inline-notification note">No user groups exist yet.</p>
			<FormLayout v-else-if="canManageGroups" @submit.prevent="saveUserGroups">
				<FormField label="Groups for this user" fake :disabled="saving">
					<CheckGroup
						v-model="userGroupIds"
						name="user-details-access-groups"
						:options="groupCheckOptions"
						:disabled="saving"
					/>
				</FormField>
				<template #actions>
					<button type="submit" class="good" :disabled="saving">Save group membership</button>
				</template>
			</FormLayout>
			<FormField v-else label="Groups for this user" fake>
				<CheckGroup
					v-model="userGroupIds"
					name="user-details-access-groups-view"
					:options="groupCheckOptions"
					disabled
				/>
			</FormField>
			<p v-if="actionMessage" class="inline-notification" :class="actionMessageType">{{ actionMessage }}</p>

			<h3 class="subsection-title">Effective roles</h3>
			<p class="section-hint">Computed from group membership. Assign roles on group detail pages.</p>
			<p v-if="!effectiveRoleNames.length" class="inline-notification note">No roles via group membership.</p>
			<p v-else>
				<span v-for="name in effectiveRoleNames" :key="name" class="role-tag">{{ name }}</span>
			</p>

			<h3 class="subsection-title">Effective permissions</h3>
			<template v-if="permissions.length">
				<p v-if="userIsSuperuser" class="inline-notification note">
					This user has the <strong>superuser</strong> role via group membership — all permissions are granted.
				</p>
				<table class="perm-audit-table">
					<thead>
						<tr>
							<th class="perm-status-col">Status</th>
							<th>Permission</th>
							<th>Granted via groups</th>
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
								<span v-if="userIsSuperuser && row.grantingGroups.length === 0" class="role-tag superuser-tag">superuser</span>
								<template v-else>
									<span v-for="gn in row.grantingGroups" :key="gn" class="role-tag">{{ gn }}</span>
									<span v-if="row.grantingGroups.length === 0" class="muted">—</span>
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
	import { HugeiconsIcon } from '@hugeicons/vue';
	import { ArrowLeft01Icon, RefreshIcon, UserGroupIcon } from '@hugeicons/core-free-icons';
	import Section from 'picocrank/vue/components/Section.vue';
	import FormLayout from 'picocrank/vue/components/FormLayout.vue';
	import FormField from 'picocrank/vue/components/FormField.vue';
	import CheckGroup from 'picocrank/vue/components/CheckGroup.vue';
	import { waitForClient } from '../javascript/util';

	const iconStrokeWidth = 2.5;
	const route = useRoute();
	const username = ref('');

	const statusPerms = ref([]);
	const statusSuper = ref(false);
	const allGroups = ref([]);
	const groupRoles = ref({});
	const roles = ref([]);
	const permissions = ref([]);
	const userGroupIds = ref([]);
	const effectiveRoleIds = ref([]);
	const loading = ref(false);
	const saving = ref(false);
	const errorMessage = ref('');
	const actionMessage = ref('');
	const actionMessageType = ref('');

	const userId = computed(() => {
		const n = parseInt(String(route.params.id), 10);
		return Number.isFinite(n) && n > 0 ? n : 0;
	});

	const pageTitle = computed(() =>
		username.value ? `${username.value} — Permissions Audit` : 'Permissions Audit',
	);

	const canViewGroups = computed(
		() => statusSuper.value || (Array.isArray(statusPerms.value) && statusPerms.value.includes('usergroups.view'))
	);
	const canViewRbac = computed(
		() => statusSuper.value || (Array.isArray(statusPerms.value) && statusPerms.value.includes('rbac.view'))
	);
	const canManageGroups = computed(
		() => statusSuper.value || (Array.isArray(statusPerms.value) && statusPerms.value.includes('usergroups.manage'))
	);

	const groupCheckOptions = computed(() =>
		allGroups.value.map((g) => ({
			value: g.id,
			label: `${g.name} (${g.memberCount} member${g.memberCount === 1 ? '' : 's'})`,
		})),
	);

	const roleById = computed(() => {
		const m = {};
		for (const r of roles.value) {
			m[r.id] = r;
		}
		return m;
	});

	const effectiveRoleNames = computed(() =>
		effectiveRoleIds.value
			.map((id) => roleById.value[id]?.name || `#${id}`)
			.sort(),
	);

	const userIsSuperuser = computed(() => effectiveRoleNames.value.includes('superuser'));

	const permissionAudit = computed(() => {
		const selectedGroups = userGroupIds.value
			.map((id) => allGroups.value.find((g) => g.id === id))
			.filter(Boolean);

		return [...permissions.value]
			.sort((a, b) => a.name.localeCompare(b.name))
			.map((p) => {
				const grantingGroups = selectedGroups
					.filter((g) => {
						const roleIds = groupRoles.value[g.id] || [];
						return roleIds.some((rid) => roleById.value[rid]?.permissionIds?.includes(p.id));
					})
					.map((g) => g.name)
					.sort();
				const granted = userIsSuperuser.value || grantingGroups.length > 0;
				return { name: p.name, granted, grantingGroups };
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

	async function load() {
		if (!userId.value) return;

		loading.value = true;
		errorMessage.value = '';
		actionMessage.value = '';
		try {
			await waitForClient();
			await refreshStatus();

			if (!canViewGroups.value && !canViewRbac.value) {
				errorMessage.value = 'You do not have permission to view user access (usergroups.view or rbac.view).';
				return;
			}

			await loadUserBasics();

			const [rolesRes, permsRes, userRolesRes] = await Promise.all([
				canViewRbac.value ? window.client.listRbacRoles({}) : Promise.resolve({ roles: [] }),
				canViewRbac.value ? window.client.listRbacPermissions({}) : Promise.resolve({ permissions: [] }),
				canViewRbac.value ? window.client.getUserRbacRoles({ userId: userId.value }) : Promise.resolve({ roleIds: [] }),
			]);

			roles.value = rolesRes.roles || [];
			permissions.value = permsRes.permissions || [];
			effectiveRoleIds.value = [...(userRolesRes.roleIds || [])];

			if (!canViewGroups.value) {
				return;
			}

			const groupsRes = await window.client.listUserGroups({});
			allGroups.value = groupsRes.groups || [];

			const roleResults = await Promise.all(
				allGroups.value.map((g) => window.client.getUserGroupRbacRoles({ groupId: g.id }))
			);
			const rolesMap = {};
			for (let i = 0; i < allGroups.value.length; i++) {
				rolesMap[allGroups.value[i].id] = roleResults[i].roleIds || [];
			}
			groupRoles.value = rolesMap;

			const memberResults = await Promise.all(
				allGroups.value.map((g) => window.client.getUserGroupMembers({ groupId: g.id }))
			);
			const ids = [];
			for (let i = 0; i < allGroups.value.length; i++) {
				if ((memberResults[i].userIds || []).includes(userId.value)) {
					ids.push(allGroups.value[i].id);
				}
			}
			userGroupIds.value = ids;
		} catch (e) {
			errorMessage.value = e.message || 'Failed to load user access.';
		} finally {
			loading.value = false;
		}
	}

	async function saveUserGroups() {
		if (!canManageGroups.value || !userId.value) return;
		saving.value = true;
		actionMessage.value = '';
		actionMessageType.value = '';
		try {
			await waitForClient();
			for (const g of allGroups.value) {
				const res = await window.client.getUserGroupMembers({ groupId: g.id });
				const current = res.userIds || [];
				const shouldInclude = userGroupIds.value.includes(g.id);
				const isIncluded = current.includes(userId.value);

				if (shouldInclude && !isIncluded) {
					await window.client.setUserGroupMembers({
						groupId: g.id,
						userIds: [...current, userId.value],
					});
				} else if (!shouldInclude && isIncluded) {
					await window.client.setUserGroupMembers({
						groupId: g.id,
						userIds: current.filter((id) => id !== userId.value),
					});
				}
			}
			actionMessage.value = 'Group membership updated.';
			actionMessageType.value = 'success';
			await load();
		} catch (e) {
			actionMessage.value = e.message || 'Failed to save group membership.';
			actionMessageType.value = 'error';
		} finally {
			saving.value = false;
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

	.section-hint {
		margin: 0 0 0.75rem;
		font-size: 0.88rem;
		opacity: 0.85;
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
