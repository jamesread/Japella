<template>
	<Section
		title="User List"
		subtitle="This page shows a list of users in the system. You cannot register new users from this panel."
		classes="settings-users"
	>
		<template #toolbar>
			<button @click="fetchUsers" class="neutral">
				<Icon icon="material-symbols:refresh" />
			</button>
		</template>

		<div v-if="users.length === 0">
			<p class="inline-notification note">No users available.</p>
		</div>
		<div v-else>
			<table>
				<thead>
					<tr>
						<th>Username</th>
						<th>Email</th>
						<th class="small" style="text-align: right">Actions</th>
					</tr>
				</thead>
				<tbody>
					<tr v-for="user in users" :key="user.id">
						<td>{{ user.username }}</td>
						<td>{{ user.email }}</td>
						<td align="right">
							<button @click="deleteUser(user.id)" class="bad">
								<Icon icon="material-symbols:delete" />
							</button>
						</td>
					</tr>
				</tbody>
			</table>
		</div>
	</Section>
</template>

<script setup>
	import { ref, onMounted } from 'vue';
	import { waitForClient } from '../javascript/util';
	import { Icon } from '@iconify/vue';
	import Section from 'picocrank/vue/components/Section.vue';

	const users = ref([]);

	async function fetchUsers() {
		await waitForClient();

		let res = await window.client.getUsers();
		users.value = res.users;
	}

	function deleteUser(userId) {
		if (confirm(`Are you sure you want to delete user with ID ${userId}?`)) {
			window.client.deleteUser({ userId })
				.then(() => {
					alert('User deleted successfully.');
					fetchUsers(); // Refresh the user list
				})
				.catch((error) => {
					console.error('Error deleting user:', error);
					alert('Failed to delete user.');
				});
		}
	}

	onMounted(() => {
		fetchUsers();
	})
</script>
