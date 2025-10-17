<template>
	<Section
		:title="account?.identity || 'Social Account'"
		:subtitle="account ? account.connector : 'Loading account details'"
		classes="social-account-details"
		:padding="false"
	>
		<template #toolbar>
			<button @click="refresh" class="neutral" :disabled="!clientReady || loading">
				<Icon icon="material-symbols:refresh" />
			</button>
			<button @click="refreshAccount" class="good" :disabled="!clientReady || loading">
				<Icon icon="material-symbols:sync" />
			</button>
			<button @click="toggleActive" :class="account?.active ? 'warning' : 'good'" :disabled="!clientReady || loading">
				<Icon :icon="account?.active ? 'material-symbols:toggle-off' : 'material-symbols:toggle-on'" />
			</button>
			<button @click="deleteAccount" class="bad" :disabled="!clientReady || loading">
				<Icon icon="material-symbols:delete" />
			</button>
		</template>

		<div v-if="!clientReady || loading">
			<p>Loading...</p>
		</div>
		<div v-else-if="error">
			<p class="inline-notification error">{{ error }}</p>
		</div>
		<div v-else-if="!account">
			<p class="inline-notification note">Account not found.</p>
		</div>
		<div v-else class="details-grid">
			<p><strong>ID:</strong> {{ account.id }}</p>
			<p><strong>Connector:</strong> {{ account.connector }}</p>
			<p><strong>Identity:</strong> {{ account.identity }}</p>
			<p><strong>Active:</strong> {{ account.active ? 'Yes' : 'No' }}</p>
		</div>
	</Section>
</template>

<script setup>
	import { ref, onMounted } from 'vue';
	import { useRoute } from 'vue-router';
	import { Icon } from '@iconify/vue';
	import Section from 'picocrank/vue/components/Section.vue';
import { useRouter } from 'vue-router';

	const route = useRoute();
const router = useRouter();
	const clientReady = ref(false)
	const loading = ref(true)
	const error = ref('')
	const accountId = ref(0)
	const account = ref(null)

	function waitForClient() {
		return new Promise((resolve) => {
			const check = () => {
				if (window.client) resolve(true); else setTimeout(check, 100);
			};
			check();
		});
	}

	async function fetchAccount() {
		loading.value = true
		error.value = ''
		try {
			const res = await window.client.getSocialAccounts({ onlyActive: false })
			const list = res.accounts || []
			account.value = list.find(a => a.id === accountId.value) || null
			if (!account.value) {
				error.value = 'Account not found.'
			}
		} catch (e) {
			error.value = `Failed to load account: ${e.message || e}`
		} finally {
			loading.value = false
		}
	}

	async function refresh() {
		await fetchAccount()
	}

	async function refreshAccount() {
		try {
			await window.client.refreshSocialAccount({ id: accountId.value })
			await fetchAccount()
		} catch (e) {
			error.value = `Failed to refresh account: ${e.message || e}`
		}
	}

async function toggleActive() {
	try {
		const next = !(account.value?.active)
		await window.client.setSocialAccountActive({ id: accountId.value, active: next })
		await fetchAccount()
	} catch (e) {
		error.value = `Failed to update active state: ${e.message || e}`
	}
}

async function deleteAccount() {
	if (!confirm('Are you sure you want to delete this account?')) return
	try {
		await window.client.deleteSocialAccount({ id: accountId.value })
		router.push({ name: 'socialAccounts' })
	} catch (e) {
		error.value = `Failed to delete account: ${e.message || e}`
	}
}

	onMounted(async () => {
		accountId.value = parseInt(route.params.id, 10) || 0
		await waitForClient()
		clientReady.value = true
		await fetchAccount()
	})
</script>

<style scoped>
	.details-grid {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 0.5rem 1rem;
	}
</style>
