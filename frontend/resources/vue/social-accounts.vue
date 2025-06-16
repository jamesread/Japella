<template>
	<section class = "social-accounts">
		<div class = "flex-row">
			<div class = "fg1">
				<h2>Social Accounts</h2>

				<p>This page shows a list of social accounts that can be used in the chat.</p>
			</div>
			<div role = "toolbar">
				<button @click = "refreshAccounts" :disabled = "!clientReady">
					<Icon icon="material-symbols:refresh" />
				</button>

				<button>
					<Icon icon="material-symbols:add-rounded" />
				</button>
			</div>
		</div>

		<div v-if = "errorMessage">
			<p class = "inline-notification error">{{ errorMessage }}</p>
		</div>
		<div v-else>
			<div v-if = "accounts.length === 0" class = "empty">
				<p class = "inline-notification note">No social accounts available.</p>
			</div>
			<div v-else class = "empty">
				<table>
					<thead>
						<tr>
							<th>ID</th>
							<th>Connector</th>
							<th>Identity</th>
							<th class="small">Actions</th>
						</tr>
					</thead>
					<tbody>
						<tr v-for="account in accounts" :key="account.id">
							<td>{{ account.id }}</td>
							<td>
								<Icon :icon="account.icon" />
								{{ account.connector }}
							</td>
							<td>{{ account.identity }}</td>
							<td>
								<button @click="refreshAccount(account.id)" class="good">
									<Icon icon="material-symbols:refresh" />
								</button>

								&nbsp;

								<button @click="deleteAccount(account.id)" class = "bad">
									<Icon icon="material-symbols:delete" />
								</button>
							</td>
						</tr>
					</tbody>
				</table>
			</div>
		</div>
	</section>
</template>

<script setup>
	import { Icon } from '@iconify/vue';
	import { ref, onMounted } from 'vue';
	import { waitForClient } from '../javascript/util';

	const posts = ref([])
	const clientReady = ref(false)
	const errorMessage = ref("")
	const accounts = ref([])

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
		window.client.refreshSocialAccount({ "id": accountId })
			.then(() => {
				refreshAccounts()
			})
			.catch((error) => {
				errorMessage.value = "Failed to refresh social account: " + error.message
				console.error('Error refreshing social account:', error)
			})
	}

	async function refreshAccounts() {
		console.log('Refreshing social accounts...')

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
</script>
