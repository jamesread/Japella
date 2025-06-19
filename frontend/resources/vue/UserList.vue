<template>
	<section class = "settings-users" title = "User List">
		<h2>User List</h2>
		<p>This page shows a list of users in the system.</p>

		<div v-if="users.length === 0">
			<p class="inline-notification note">No users available.</p>
		</div>
		<div v-else>
			<table>
				<thead>
					<tr>
						<th>Username</th>
						<th>Email</th>
						<th>Actions</th>
					</tr>
				</thead>

				<tbody>
					<tr v-for="user in users" :key="user.id">
						<td>{{ user.username }}</td>
						<td>{{ user.email }}</td>
						<td>
							<button @click="deleteUser(user.id)" class="bad">Delete</button>
						</td>
					</tr>
				</tbody>
			</table>
		</div>

		<p>You cannot register new users from this panel.</p>
	</section>
</template>

<script setup>
	import { ref, onMounted } from 'vue';
	import { waitForClient } from '../javascript/util';

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
