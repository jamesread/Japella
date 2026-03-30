<template>
	<Section
		title="User List"
		subtitle="View users, create new accounts, or remove users you no longer need. You cannot delete your own account while signed in."
		classes="settings-users"
		:padding="false"
	>
		<template #toolbar>
			<router-link :to="{ name: 'createUser' }" class="button good">
				<Icon icon="material-symbols:person-add" />
				Create user
			</router-link>
			<button type="button" @click="fetchUsers" class="neutral" title="Refresh">
				<Icon icon="material-symbols:refresh" />
			</button>
		</template>

		<div v-if="users.length === 0" class="user-table-wrap">
			<p class="inline-notification note">No users available.</p>
		</div>
		<Table
			v-else
			class="user-table-wrap"
			row-clickable
			:data="users"
			:headers="tableHeaders"
			:show-pagination="false"
			@row-click="openUserDetails"
		>
			<template #cell-username="{ row, value }">
				<router-link
					:to="{ name: 'userDetails', params: { id: String(row.id) } }"
					class="username-link"
				>
					{{ value }}
				</router-link>
			</template>
			<template #cell-actions="{ row }">
				<div class="actions-cell">
					<button
						type="button"
						class="bad"
						:disabled="deleting || isCurrentUser(row)"
						:title="isCurrentUser(row) ? 'You cannot delete your own account' : 'Delete user'"
						@click="deleteUserAccount(row)"
					>
						<Icon icon="material-symbols:delete" />
					</button>
				</div>
			</template>
		</Table>
	</Section>
</template>

<script setup>
	import { ref, onMounted } from 'vue';
	import { useRouter } from 'vue-router';
	import { waitForClient } from '../javascript/util';
	import { Icon } from '@iconify/vue';
	import Section from 'picocrank/vue/components/Section.vue';
	import Table from './picocrank/TableWithRowClick.vue';

	const router = useRouter();

	const tableHeaders = [
		{ key: 'username', label: 'Username', sortable: true },
		{ key: 'createdAt', label: 'Created', sortable: true },
		{ key: 'actions', label: 'Actions', sortable: false, width: '6rem' },
	];

	const users = ref([]);
	const currentUsername = ref('');
	const deleting = ref(false);

	function isCurrentUser(user) {
		return Boolean(currentUsername.value && user.username === currentUsername.value);
	}

	function openUserDetails(row) {
		router.push({ name: 'userDetails', params: { id: String(row.id) } });
	}

	async function refreshCurrentUsername() {
		await waitForClient();
		const st = await window.client.getStatus({});
		currentUsername.value = st.isLoggedIn ? st.username : '';
	}

	async function fetchUsers() {
		await waitForClient();
		const res = await window.client.getUsers();
		users.value = res.users || [];
	}

	async function deleteUserAccount(user) {
		if (isCurrentUser(user)) {
			return;
		}
		if (!confirm(`Delete user "${user.username}"? This cannot be undone.`)) {
			return;
		}
		deleting.value = true;
		try {
			await waitForClient();
			const res = await window.client.deleteUser({ userId: user.id });
			if (res.standardResponse?.success) {
				await fetchUsers();
			} else {
				alert(res.standardResponse?.message || 'Failed to delete user.');
			}
		} catch (error) {
			console.error('Error deleting user:', error);
			alert(error.message || 'Failed to delete user.');
		} finally {
			deleting.value = false;
		}
	}

	onMounted(async () => {
		await waitForClient();
		await Promise.all([fetchUsers(), refreshCurrentUsername()]);
	});
</script>

<style scoped>
	.user-table-wrap {
		margin-top: 0.5rem;
	}

	.actions-cell {
		text-align: right;
	}

	.username-link {
		color: inherit;
		font-weight: 600;
		text-decoration: underline;
		text-underline-offset: 0.15em;
	}

	.username-link:hover {
		opacity: 0.88;
	}
</style>
