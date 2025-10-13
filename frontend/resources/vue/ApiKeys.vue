<script setup>
	import { ref, onMounted } from 'vue';
	import { waitForClient } from '../javascript/util';
	import { Icon } from '@iconify/vue';
    import Section from 'picocrank/vue/components/Section.vue';

	const apiKeys = ref([]);
	const newKeyDialog = ref(null);
	const newKeyValue = ref('');

	async function fetchApiKeys() {
		await waitForClient();

		let res = await window.client.getApiKeys();
		apiKeys.value = res.keys;
	}

	async function revokeKey(keyId) {
		if (confirm(`Are you sure you want to revoke the API key with ID ${keyId}?`)) {
			try {
				await window.client.revokeApiKey({
					id: keyId
				});
				alert('API key revoked successfully.');
				fetchApiKeys(); // Refresh the API keys list
			} catch (error) {
				console.error('Error revoking API key:', error);
				alert('Failed to revoke API key.');
			}
		}
	}

	function showNewKeyDialog(keyValue) {
		newKeyValue.value = keyValue;
		newKeyDialog.value.showModal();
	}

	function createNewKey() {
		window.client.createApiKey()
			.then((res) => {
				showNewKeyDialog(res.newKeyValue);
				fetchApiKeys(); // Refresh the API keys list
			})
			.catch((error) => {
				console.error('Error creating API key:', error);
				alert('Failed to create API key.');
			});
	}

	onMounted(() => {
		fetchApiKeys();
	});
</script>

<template>
	<dialog ref = "newKeyDialog" class = "dialog">
		<h2>Your new key</h2>
		<p>Here is your new API key. Please copy it and store it securely, as you will not be able to see it again.</p>

		<form action="#" method="dialog">
			<input type="text" readonly class="key-input" :value="newKeyValue" />
			<button type="submit" class="good">OK</button>
		</form>
    </dialog>
    <Section
        title="API Keys"
        subtitle="View, revoke, and create API keys. Use Bearer token in Authorization header."
        :padding="false"
        classes="api-keys"
    >
        <template #toolbar>
            <button @click="fetchApiKeys" class="neutral">
                <Icon icon="material-symbols:refresh" />
            </button>

            <button @click="createNewKey" class="neutral">
                <Icon icon="material-symbols:add" />
            </button>
        </template>

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
    </Section>
</template>

<style scoped>
    form {
		grid-template-columns: 1fr;
	}

	input {
		text-align: center;
	}

	button {
		place-self: start;
	}
</style>
