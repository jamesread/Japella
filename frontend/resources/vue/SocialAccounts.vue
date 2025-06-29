<template>
	<section class = "social-accounts">
		<div class = "section-header">
			<div class = "fg1">
				<h2>Social Accounts</h2>

				<p>This page shows a list of social accounts that you have configured.</p>
			</div>
			<div role = "toolbar">
				<button @click = "refreshAccounts()" :disabled = "!clientReady" class = "neutral">
					<Icon icon="material-symbols:refresh" />
				</button>
			</div>
		</div>

		<div v-if = "errorMessage">
			<p class = "inline-notification error">{{ errorMessage }}</p>
		</div>
		<div v-else>
			<div v-if = "accounts.length === 0" class = "empty">
				<p class = "inline-notification note">No social accounts connected yet.</p>
			</div>
			<div v-else class = "empty">
				<table>
					<thead>
						<tr>
							<th>Identity</th>
							<th class="small" style = "text-align: right">Actions</th>
						</tr>
					</thead>
					<tbody>
						<tr v-for="account in accounts" :key="account.id">
							<td>
								<span class = "social-account">
									<Icon :icon="account.icon" />
									{{ account.identity }}
								</span>
							</td>
							<td align = "right">
								<button @click="refreshAccount(account.id)" class="good">
									<Icon icon="material-symbols:refresh" />
								</button>

								&nbsp;

								<button @click="setAccountActive(account.id, true)" class="good" v-if="!account.active">
									Enable
									<Icon icon="material-symbols:toggle-on" />
								</button>
								<button @click="setAccountActive(account.id, false)" class="warning" v-else>
									Disable
									<Icon icon="material-symbols:toggle-off" />
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

	<OAuthServices />
</template>

<script setup>
	import { Icon } from '@iconify/vue';
	import { ref, onMounted, inject } from 'vue';
	import { waitForClient } from '../javascript/util';

	const clientReady = ref(false)
	const errorMessage = ref("")
	const accounts = ref([])
	const showErrorDialog = inject('showSectionError')

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
			.then((ret) => {
				if (!ret.standardResponse.success) {
				    console.log('Error refreshing social account:', ret.standardResponse.message)
				    showErrorDialog?.(ret.standardResponse.message)
				}

				refreshAccounts()
			})
			.catch((error) => {
				errorMessage.value = "Failed to refresh social account: " + error.message
				console.error('Error refreshing social account:', error)
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

	function setAccountActive(accountId, active) {
		window.client.setSocialAccountActive({ "id": accountId, "active": active })
			.then(() => {
				refreshAccounts()
			})
			.catch((error) => {
				errorMessage.value = "Failed to set social account active state: " + error.message
				console.error('Error setting social account active state:', error)
			})
	}

	onMounted(async () => {
		await waitForClient()

		clientReady.value = true

		refreshAccounts()
	})
</script>
