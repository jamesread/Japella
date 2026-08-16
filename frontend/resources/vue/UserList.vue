<template>
	<Section
		subtitle="View users, create new accounts, or remove users you no longer need. You cannot delete your own account while signed in."
		classes="user-list settings-users"
		:padding="false"
	>
		<template #title>
			<span class="section-title-with-icon">
				<HugeiconsIcon :icon="UserMultiple02Icon" width="22" height="22" aria-hidden="true" />
				Users
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
				v-if="canCreate"
				type="button"
				class="inline-icon good"
				aria-label="Create user"
				title="Create user"
				:disabled="loading"
				@click="goCreate"
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
		<div v-if="loading && !users.length" class="list-banner-pad muted">Loading…</div>

		<template v-else>
			<p v-if="!users.length" class="inline-notification note list-banner-pad">No users available.</p>

			<Table
				v-else
				class="user-table-wrap"
				row-clickable
				:data="tableRows"
				:headers="tableHeaders"
				@row-click="openUserDetails"
			>
				<template #cell-username="{ value }">
					<strong>{{ value }}</strong>
				</template>
				<template #cell-actions="{ row }">
					<div v-if="canDelete" class="actions-cell">
						<button
							type="button"
							class="bad small"
							:disabled="saving || isCurrentUser(row)"
							:title="isCurrentUser(row) ? 'You cannot delete your own account' : 'Delete user'"
							@click="deleteUserAccount(row)"
						>
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
	import { Add01Icon, RefreshIcon, UserMultiple02Icon } from '@hugeicons/core-free-icons';
	import Section from 'picocrank/vue/components/Section.vue';
	import Table from './picocrank/TableWithRowClick.vue';

	const iconStrokeWidth = 2.5;
	const router = useRouter();

	const users = ref([]);
	const loading = ref(true);
	const saving = ref(false);
	const errorMessage = ref('');
	const currentUsername = ref('');
	const statusPerms = ref([]);
	const statusSuper = ref(false);

	const canCreate = computed(
		() => statusSuper.value || (Array.isArray(statusPerms.value) && statusPerms.value.includes('users.create')),
	);
	const canDelete = computed(
		() => statusSuper.value || (Array.isArray(statusPerms.value) && statusPerms.value.includes('users.delete')),
	);

	const tableHeaders = computed(() => {
		const headers = [
			{ key: 'username', label: 'Username', sortable: true },
			{ key: 'createdByLabel', label: 'Created by', sortable: true },
			{ key: 'createdAtLabel', label: 'Created', sortable: true, width: '11rem' },
		];
		if (canDelete.value) {
			headers.push({ key: 'actions', label: 'Actions', sortable: false, width: '6rem' });
		}
		return headers;
	});

	function formatDate(dateString) {
		if (!dateString) return '—';
		try {
			return new Date(dateString).toLocaleDateString();
		} catch {
			return dateString;
		}
	}

	const tableRows = computed(() =>
		users.value.map((u) => ({
			...u,
			createdByLabel: u.createdBy || u.created_by || '—',
			createdAtLabel: formatDate(u.createdAt || u.created_at),
		})),
	);

	function isCurrentUser(user) {
		return Boolean(currentUsername.value && user.username === currentUsername.value);
	}

	function goCreate() {
		router.push({ name: 'createUser' });
	}

	function openUserDetails(row) {
		router.push({ name: 'userDetails', params: { id: String(row.id) } });
	}

	async function refreshStatus() {
		await waitForClient();
		const st = await window.client.getStatus({});
		statusPerms.value = st.rbacPermissions || [];
		statusSuper.value = Boolean(st.rbacIsSuperuser);
		currentUsername.value = st.isLoggedIn ? st.username : '';
	}

	async function loadAll() {
		loading.value = true;
		errorMessage.value = '';
		try {
			await waitForClient();
			await refreshStatus();
			const res = await window.client.getUsers();
			users.value = res.users || [];
		} catch (e) {
			console.error(e);
			errorMessage.value = e?.message || 'Failed to load users';
		} finally {
			loading.value = false;
		}
	}

	async function deleteUserAccount(user) {
		if (!canDelete.value || isCurrentUser(user)) {
			return;
		}
		if (!confirm(`Delete user "${user.username}"? This cannot be undone.`)) {
			return;
		}
		saving.value = true;
		errorMessage.value = '';
		try {
			await waitForClient();
			const res = await window.client.deleteUser({ userId: user.id });
			if (res.standardResponse?.success) {
				await loadAll();
			} else {
				errorMessage.value = res.standardResponse?.message || 'Failed to delete user.';
			}
		} catch (e) {
			console.error('Error deleting user:', e);
			errorMessage.value = e?.message || 'Failed to delete user.';
		} finally {
			saving.value = false;
		}
	}

	onMounted(loadAll);
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

	.user-table-wrap {
		margin-top: 0.5rem;
		margin-bottom: 1.5rem;
	}

	.actions-cell {
		text-align: right;
	}
</style>
