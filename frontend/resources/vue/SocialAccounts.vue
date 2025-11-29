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
						<button @click="openProfile(account)" class="neutral" :title="'Open ' + account.identity + ' profile'">
							<Icon icon="material-symbols:open-in-new" />
						</button>
						&nbsp;
						<button @click="refreshAccount(account.id)" class="good" :disabled="isAccountRefreshing(account.id)">
							<Icon v-if="isAccountSuccess(account.id)" icon="material-symbols:check-circle" />
							<Icon v-else-if="isAccountRefreshing(account.id)" icon="material-symbols:hourglass-top" />
							<Icon v-else icon="material-symbols:refresh" />
						</button>
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

	function getProfileUrl(account) {
		if (!account || !account.identity) {
			return null
		}

		const identity = account.identity
		const connector = account.connector

		switch (connector) {
			case 'bluesky':
				// Bluesky profile URL format: https://bsky.app/profile/{handle}
				return `https://bsky.app/profile/${identity}`
			case 'x':
				// X/Twitter profile URL format: https://x.com/{username}
				return `https://x.com/${identity}`
			case 'mastodon':
				// Mastodon profile URL format: {homeserver}/@{username}
				// Note: We don't have homeserver in the account object from the API
				// Default to mastodon.social, but ideally we'd get this from the account
				const homeserver = 'https://mastodon.social' // Default, could be improved
				return `${homeserver}/@${identity}`
			default:
				return null
		}
	}

	function openProfile(account) {
		const profileUrl = getProfileUrl(account)
		if (profileUrl) {
			window.open(profileUrl, '_blank', 'noopener,noreferrer')
		} else {
			showErrorDialog?.('Unable to determine profile URL for this account type.')
		}
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
				    
				    // Check if the error indicates re-authentication is required
				    const errorMessage = ret.standardResponse.message.toLowerCase()
				    if (errorMessage.includes('re-authentication required') || errorMessage.includes('reauthentication required')) {
				    	// Find the account to get its connector
				    	const account = accounts.value.find(a => a.id === accountId)
				    	if (account && account.connector) {
				    		// Start OAuth flow for re-authentication
				    		console.log('Starting OAuth flow for re-authentication:', account.connector)
				    		window.client.startOAuth({ connectorId: account.connector })
				    			.then((oauthRes) => {
				    				console.log('OAuth URL received, redirecting...')
				    				window.location.href = oauthRes.url
				    			})
				    			.catch((oauthError) => {
				    				console.error('Error starting OAuth flow:', oauthError)
				    				showErrorDialog?.('Failed to start re-authentication. Please try connecting the account again from the OAuth Services section.')
				    				refreshingAccounts.value.delete(accountId)
				    			})
				    		// Note: We don't delete from refreshingAccounts here because we're redirecting
				    		return // Don't show error dialog, we're redirecting
				    	} else {
				    		showErrorDialog?.('Re-authentication required, but could not determine connector. Please reconnect the account from the OAuth Services section.')
				    	}
				    } else {
				    	showErrorDialog?.(ret.standardResponse.message)
				    }
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

<style scoped>
	.social-account {
		color: #fff;
		text-decoration: none;
	}

	.social-account:hover {
		text-decoration: underline;
	}
		
</style>