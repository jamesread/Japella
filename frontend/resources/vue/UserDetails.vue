<template>
	<Section
		:title="user?.username || 'User'"
		subtitle="Account details for this user"
		classes="user-details"
	>
		<template #toolbar>
			<router-link :to="{ name: 'settingsUsers' }" class="button neutral">
				<Icon icon="material-symbols:arrow-back" />
				Back to Users
			</router-link>
			<button type="button" class="neutral" title="Refresh" :disabled="loading" @click="load">
				<Icon icon="material-symbols:refresh" />
			</button>
			<button
				v-if="canImpersonate"
				type="button"
				class="neutral"
				title="Login as this user"
				:disabled="impersonating"
				@click="impersonate"
			>
				<Icon icon="mdi:account-switch" />
				{{ impersonating ? 'Switching…' : 'Impersonate' }}
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
			</dl>

			<div v-if="showUserAdminNav" class="user-admin-nav">
				<h3 class="subsection-title">Account administration</h3>
				<Navigation ref="userSubNav">
					<NavigationGrid />
				</Navigation>
			</div>

			<template v-if="canViewGroups">
				<h3 class="subsection-title">User groups</h3>
				<p v-if="groupsLoading" class="keys-status">Loading groups…</p>
				<p v-else-if="!allGroups.length" class="inline-notification note">No user groups exist yet.</p>
				<template v-else>
					<ul class="group-membership-list">
						<li v-for="g in allGroups" :key="g.id">
							<label class="check-label">
								<input
									v-model="userGroupIds"
									type="checkbox"
									:value="g.id"
									:disabled="!canManageGroups || groupsSaving"
								/>
								{{ g.name }}
								<span class="group-count">({{ g.memberCount }} member{{ g.memberCount === 1 ? '' : 's' }})</span>
							</label>
						</li>
					</ul>
					<button
						v-if="canManageGroups"
						type="button"
						class="good"
						:disabled="groupsSaving"
						@click="saveUserGroups"
					>
						Save group membership
					</button>
					<p v-if="groupsSaveMessage" class="inline-notification" :class="groupsSaveType">{{ groupsSaveMessage }}</p>
				</template>
			</template>
		</template>
	</Section>
</template>

<script setup>
	import { ref, computed, watch, nextTick } from 'vue';
	import { useRoute } from 'vue-router';
	import { Icon } from '@iconify/vue';
	import Section from 'picocrank/vue/components/Section.vue';
	import Navigation from 'picocrank/vue/components/Navigation.vue';
	import NavigationGrid from 'picocrank/vue/components/NavigationGrid.vue';
	import { waitForClient } from '../javascript/util';

	const route = useRoute();
	const user = ref(null);
	const loading = ref(true);
	const error = ref('');
	const impersonating = ref(false);

	const statusPerms = ref([]);
	const statusSuper = ref(false);

	const allGroups = ref([]);
	const userGroupIds = ref([]);
	const groupsLoading = ref(false);
	const groupsSaving = ref(false);
	const groupsSaveMessage = ref('');
	const groupsSaveType = ref('');

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

	const showUserAdminNav = computed(() => canManageRoles.value || canResetPassword.value || canViewApiKeys.value);

	const canImpersonate = computed(
		() => statusSuper.value || (Array.isArray(statusPerms.value) && statusPerms.value.includes('system.impersonate'))
	);

	const canViewGroups = computed(
		() => statusSuper.value || (Array.isArray(statusPerms.value) && statusPerms.value.includes('usergroups.view'))
	);

	const canManageGroups = computed(
		() => statusSuper.value || (Array.isArray(statusPerms.value) && statusPerms.value.includes('usergroups.manage'))
	);

	const userId = computed(() => {
		const n = parseInt(String(route.params.id), 10);
		return Number.isFinite(n) && n > 0 ? n : 0;
	});

	function setupUserSubNavLinks() {
		const nav = userSubNav.value;
		if (!nav || !userId.value) {
			return;
		}
		nav.clearNavigationLinks();
		if (canManageRoles.value) {
			nav.addRouterLink('userDetailsRoles', null, {
				params: { id: String(userId.value) },
				description: 'Assign RBAC roles and review effective permissions',
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
		} catch (e) {
			console.error(e);
		} finally {
			groupsLoading.value = false;
		}
	}

	async function saveUserGroups() {
		if (!canManageGroups.value || !userId.value) return;
		groupsSaving.value = true;
		groupsSaveMessage.value = '';
		groupsSaveType.value = '';
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
			groupsSaveMessage.value = 'Group membership updated.';
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

		// Navigation mounts only after loading=false; an earlier watch can run while still on the loading branch (userSubNav null).
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

	watch([showUserAdminNav, canManageRoles, canResetPassword, canViewApiKeys, userId, loading], () => {
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

	.check-label {
		display: block;
		margin: 0.25rem 0;
		font-size: 0.9em;
	}

	.group-membership-list {
		margin: 0 0 0.75rem;
		padding: 0;
		list-style: none;
		max-width: 40rem;
	}

	.group-count {
		opacity: 0.7;
		font-size: 0.85em;
	}
</style>
