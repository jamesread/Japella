<template>
	<Section
		title="Social Accounts"
		subtitle="This page shows a list of social accounts that you have configured."
		classes="social-accounts"
		:padding="false"
	>
		<template #toolbar>
			<button @click="refreshAccounts()" :disabled="!clientReady" class="neutral">
				<Icon icon="material-symbols:refresh" />
			</button>
		</template>

		<div v-if="errorMessage">
			<p class="inline-notification error">{{ errorMessage }}</p>
		</div>
		<div v-else>
			<div v-if="accounts.length === 0">
				<p class="inline-notification note">No social accounts connected yet.</p>
			</div>
			<table class = "data-table" v-else>
				<thead>
					<tr>
						<th>Identity</th>
						<th class="actions" style="text-align: right">Actions</th>
					</tr>
				</thead>
				<tbody>
					<tr v-for="account in accounts" :key="account.id">
					<td>
						<router-link :to="{ name: 'socialAccountDetails', params: { id: account.id } }" class="social-account">
							<Icon :icon="account.icon" />
							{{ account.identity }}
						</router-link>
					</td>
						<td align="right">
						<button @click="refreshAccount(account.id)" class="good" :disabled="isAccountRefreshing(account.id)">
							<Icon v-if="isAccountSuccess(account.id)" icon="material-symbols:check-circle" />
							<Icon v-else-if="isAccountRefreshing(account.id)" icon="material-symbols:hourglass-top" />
							<Icon v-else icon="material-symbols:refresh" />
						</button>

							&nbsp;




						</td>
					</tr>
				</tbody>
			</table>
		</div>
	</Section>

	<OAuthServices />
</template>

<script setup>
	import { Icon } from '@iconify/vue';
	import { ref, onMounted, inject, onActivated } from 'vue';
	import { waitForClient } from '../javascript/util';
	import Section from 'picocrank/vue/components/Section.vue';

	const clientReady = ref(false)
	const errorMessage = ref("")
	const accounts = ref([])
	const showErrorDialog = inject('showSectionError')
	const refreshingAccounts = ref(new Set())
	const successAccounts = ref(new Set())

	function isAccountRefreshing(accountId) {
		return refreshingAccounts.value.has(accountId)
	}

	function isAccountSuccess(accountId) {
		return successAccounts.value.has(accountId)
	}

	function deleteAccount(accountId) {
		if (!confirm("Are you sure you want to delete this account?")) {
			return
		}

		window.client.deleteSocialAccount({ "id": accountId })
			.then(() => {
				refreshAccounts()
			})
			.catch((error) => {
				errorMessage.value = "Failed to delete social account: " + error.message
				console.error('Error deleting social account:', error)
			})
	}

	function refreshAccount(accountId) {
		refreshingAccounts.value.add(accountId)

		window.client.refreshSocialAccount({ "id": accountId })
			.then((ret) => {
				if (!ret.standardResponse.success) {
				    console.log('Error refreshing social account:', ret.standardResponse.message)
				    showErrorDialog?.(ret.standardResponse.message)
				}

				// Mark success briefly when the refresh call succeeds at the transport level
				if (ret.standardResponse.success) {
					successAccounts.value.add(accountId)
					setTimeout(() => {
						successAccounts.value.delete(accountId)
					}, 1500)
				}

				refreshAccounts()
			})
			.catch((error) => {
				errorMessage.value = "Failed to refresh social account: " + error.message
				console.error('Error refreshing social account:', error)
			})
			.finally(() => {
				refreshingAccounts.value.delete(accountId)
			})
	}

	async function refreshAccounts() {
		return await window.client.getSocialAccounts()
			.then((ret) => {
				ret.accounts.sort((a, b) => Number(a.id) - Number(b.id))

				accounts.value = ret.accounts || []
			})
			.catch((error) => {
				errorMessage.value = "Failed to fetch social accounts: " + error.message
				console.error('Error fetching social accounts:', error)
				return []
			})
	}



	onMounted(async () => {
		await waitForClient()

		clientReady.value = true

		refreshAccounts()
	})

	onActivated(() => {
		refreshAccounts()
	})
</script>
