<script setup>
	import { ref, computed, watch } from 'vue';
	import { useRoute } from 'vue-router';
	import { waitForClient } from '../javascript/util';
	import { Icon } from '@iconify/vue';
	import Section from 'picocrank/vue/components/Section.vue';

	const route = useRoute();
	const allKeys = ref([]);
	const newKeyDialog = ref(null);
	const newKeyValue = ref('');
	const targetUser = ref(null);

	const filterUserId = computed(() => {
		const n = parseInt(String(route.params.id), 10);
		return Number.isFinite(n) && n > 0 ? n : 0;
	});

	const apiKeys = computed(() => {
		if (!filterUserId.value) return allKeys.value;
		return allKeys.value.filter((k) => k.userId === filterUserId.value);
	});

	const sectionTitle = computed(() => {
		if (targetUser.value) return `API Keys — ${targetUser.value.username}`;
		if (filterUserId.value) return `API Keys — User #${filterUserId.value}`;
		return 'API Keys';
	});

	const sectionSubtitle = computed(() => {
		if (filterUserId.value) return 'View and manage API keys for this user. Use Bearer token in Authorization header.';
		return 'View, revoke, and create API keys. Use Bearer token in Authorization header.';
	});

	async function fetchApiKeys() {
		await waitForClient();

		let res = await window.client.getApiKeys();
		allKeys.value = res.keys;
	}

	async function loadTargetUser() {
		if (!filterUserId.value) {
			targetUser.value = null;
			return;
		}
		try {
			await waitForClient();
			const res = await window.client.getUser({ userId: filterUserId.value });
			targetUser.value = res.user ?? null;
		} catch {
			targetUser.value = null;
		}
	}

	async function revokeKey(keyId) {
		if (confirm(`Are you sure you want to revoke the API key with ID ${keyId}?`)) {
			try {
				await window.client.revokeApiKey({
					id: keyId
				});
				alert('API key revoked successfully.');
				fetchApiKeys();
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
				fetchApiKeys();
			})
			.catch((error) => {
				console.error('Error creating API key:', error);
				alert('Failed to create API key.');
			});
	}

	watch(filterUserId, () => {
		loadTargetUser();
		fetchApiKeys();
	}, { immediate: true });
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
        :title="sectionTitle"
        :subtitle="sectionSubtitle"
        :padding="false"
        classes="api-keys"
    >
        <template #toolbar>
            <router-link v-if="filterUserId" :to="{ name: 'userDetails', params: { id: String(filterUserId) } }" class="button neutral">
                <Icon icon="material-symbols:arrow-back" />
                Back to User
            </router-link>
            <button @click="fetchApiKeys" class="neutral">
                <Icon icon="material-symbols:refresh" />
            </button>

            <button v-if="!filterUserId" @click="createNewKey" class="neutral">
                <Icon icon="material-symbols:add" />
            </button>
        </template>

        <p v-if="apiKeys.length === 0" class="inline-notification note">No API keys available.</p>
        <table v-else>
            <thead>
                <tr>
                    <th>ID</th>
                    <th>Key</th>
                    <th v-if="!filterUserId">User Account</th>
                    <th>Created At</th>
                    <th></th>
                </tr>
            </thead>
            <tbody>
                <tr v-for="key in apiKeys" :key="key.id">
                    <td>{{ key.id }}</td>
                    <td>{{ key.keyValue }}</td>
                    <td v-if="!filterUserId">{{ key.username }}</td>
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
