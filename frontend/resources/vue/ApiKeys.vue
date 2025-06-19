<script setup>
	import { ref, onMounted } from 'vue';
	import { waitForClient } from '../javascript/util';
	import { Icon } from '@iconify/vue';

	const apiKeys = ref([]);

	async function fetchApiKeys() {
		await waitForClient();

		let res = await window.client.getApiKeys();
		apiKeys.value = res.keys;
	}

	async function revokeKey(keyId) {
		if (confirm(`Are you sure you want to revoke the API key with ID ${keyId}?`)) {
			try {
				await window.client.revokeApiKey({ keyId });
				alert('API key revoked successfully.');
				fetchApiKeys(); // Refresh the API keys list
			} catch (error) {
				console.error('Error revoking API key:', error);
				alert('Failed to revoke API key.');
			}
		}
	}

	onMounted(() => {
		fetchApiKeys();
	});
</script>

<template>
	<section>
		<div class = "section-header">
			<div class = "fg1">
				<h2>API Keys</h2>

				<p>This page allows you to manage your API keys. You can revoke existing keys.</p>
			</div>
			<div role = "toolbar">
				<button @click="fetchApiKeys" class="neutral">
					<Icon icon="material-symbols:refresh" />
				</button>
			</div>
		</div>

		<p v-if="apiKeys.length === 0" class="inline-notification note">No API keys available.</p>
		<table v-else>
			<thead>
				<tr>
					<th>ID</th>
					<th>Key</th>
					<th>User Account</th>
					<th>Created At</th>
					<th></th>
				</tr>
			</thead>
			<tbody>
				<tr v-for="key in apiKeys" :key="key.id">
					<td>{{ key.id }}</td>
					<td>{{ key.keyValue }}</td>
					<td>{{ key.username }}</td>
					<td>{{ new Date(key.createdAt).toLocaleString() }}</td>
					<td align="right">
						<button @click="revokeKey(key.id)" class="bad">Revoke</button>
					</td>
				</tr>
			</tbody>
		</table>
	</section>
</template>
