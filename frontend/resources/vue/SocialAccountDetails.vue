<template>
	<Section
		:title="account?.identity || 'Social Account'"
		:subtitle="account ? account.connector : 'Loading account details'"
		classes="social-account-details"
		:padding="false"
	>
		<template #toolbar>
			<button v-if="getProfileUrl(account)" @click="openProfile" class="neutral" :disabled="!clientReady || loading" title="Open Profile">
				<Icon icon="mdi:open-in-new" />
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
		<dl v-else>
			<dt>ID</dt>
			<dd>{{ account.id }}</dd>
			<dt>Connector</dt>
			<dd>{{ account.connector }}</dd>
			<dt>Identity</dt>
			<dd>
				<span>{{ account.identity }}</span>
				<button
					v-if="getProfileUrl(account)"
					@click="openProfile"
					class="profile-link-button"
					:title="'Open ' + account.identity + ' profile'"
				>
					<Icon icon="mdi:open-in-new" width="16" height="16" />
				</button>
			</dd>
			<dt>Active</dt>
			<dd>{{ account.active ? 'Yes' : 'No' }}</dd>
			<dt>Last Posted</dt>
			<dd>{{ lastPosted || 'Never' }}</dd>
			<dt>Token Expiry</dt>
			<dd>
				<span v-if="account.tokenExpiry" :title="formatAbsoluteDate(account.tokenExpiry)">{{ formatRelativeTime(account.tokenExpiry) }}</span>
				<span v-else>{{ formatRelativeTime(null) }}</span>
			</dd>
		</dl>
	</Section>

	<Section
		v-if="!loading && account"
		title="Account Actions"
		subtitle="Manage this social account"
	>
		<div class="action-item">
			<div class="action-description">
				<Icon icon="material-symbols:sync" width="24" height="24" />
				<div>
					<h5 style="margin: 0 0 0.25em 0;">Refresh Account</h5>
					<p style="margin: 0; font-size: 0.9em;">Sync and refresh the account with the social media service.</p>
				</div>
			</div>
			<button @click="refreshAccount" class="good" :disabled="!clientReady || loading">
				<Icon icon="material-symbols:sync" width="16" height="16" />
				<span>Refresh</span>
			</button>
		</div>

		<div class="action-item">
			<div class="action-description">
				<Icon :icon="account?.active ? 'material-symbols:toggle-off' : 'material-symbols:toggle-on'" width="24" height="24" />
				<div>
					<h5 style="margin: 0 0 0.25em 0;">Toggle Active Status</h5>
					<p style="margin: 0; font-size: 0.9em;">{{ account?.active ? 'Deactivate this account to prevent it from being used for posting.' : 'Activate this account to enable posting.' }}</p>
				</div>
			</div>
			<button @click="toggleActive" :class="account?.active ? 'warning' : 'good'" :disabled="!clientReady || loading">
				<Icon :icon="account?.active ? 'material-symbols:toggle-off' : 'material-symbols:toggle-on'" width="16" height="16" />
				<span>{{ account?.active ? 'Deactivate' : 'Activate' }}</span>
			</button>
		</div>

		<div class="action-item">
			<div class="action-description">
				<Icon icon="material-symbols:delete" width="24" height="24" />
				<div>
					<h5 style="margin: 0 0 0.25em 0;">Delete Account</h5>
					<p style="margin: 0; font-size: 0.9em;">Permanently remove this social account. This action cannot be undone.</p>
				</div>
			</div>
			<button @click="deleteAccount" class="bad" :disabled="!clientReady || loading">
				<Icon icon="material-symbols:delete" width="16" height="16" />
				<span>Delete</span>
			</button>
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
	const lastPosted = ref(null)

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
			} else {
				await fetchLastPosted()
			}
		} catch (e) {
			error.value = `Failed to load account: ${e.message || e}`
		} finally {
			loading.value = false
		}
	}

	async function fetchLastPosted() {
		if (!accountId.value) return

		try {
			const timelineRes = await window.client.getTimeline()
			const posts = timelineRes.posts || []

			// Filter posts for this social account and find the most recent one
			const accountPosts = posts.filter(post => post.socialAccountId === accountId.value)

			if (accountPosts.length > 0) {
				// Sort by created date (most recent first) and get the first one
				const sortedPosts = accountPosts.sort((a, b) => {
					const dateA = new Date(a.created)
					const dateB = new Date(b.created)
					return dateB - dateA
				})
				lastPosted.value = sortedPosts[0].created
			} else {
				lastPosted.value = null
			}
		} catch (e) {
			console.error('Error fetching last posted date:', e)
			lastPosted.value = null
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

	function openProfile() {
		const profileUrl = getProfileUrl(account.value)
		if (profileUrl) {
			window.open(profileUrl, '_blank', 'noopener,noreferrer')
		}
	}

	function formatRelativeTime(dateString) {
		if (!dateString) {
			return 'Never expires'
		}

		try {
			const expiryDate = new Date(dateString)
			if (isNaN(expiryDate.getTime())) {
				return 'Invalid date'
			}

			const now = new Date()
			const diffMs = expiryDate - now
			const diffSeconds = Math.floor(Math.abs(diffMs) / 1000)
			const diffMinutes = Math.floor(diffSeconds / 60)
			const diffHours = Math.floor(diffMinutes / 60)
			const diffDays = Math.floor(diffHours / 24)
			const diffWeeks = Math.floor(diffDays / 7)
			const diffMonths = Math.floor(diffDays / 30)
			const diffYears = Math.floor(diffDays / 365)

			const isPast = diffMs < 0
			const prefix = isPast ? '' : 'In '
			const suffix = isPast ? ' ago' : ''

			if (diffSeconds < 60) {
				return isPast ? 'Just expired' : 'Expires soon'
			} else if (diffMinutes < 60) {
				return `${prefix}${diffMinutes} minute${diffMinutes !== 1 ? 's' : ''}${suffix}`
			} else if (diffHours < 24) {
				return `${prefix}${diffHours} hour${diffHours !== 1 ? 's' : ''}${suffix}`
			} else if (diffDays < 7) {
				return `${prefix}${diffDays} day${diffDays !== 1 ? 's' : ''}${suffix}`
			} else if (diffWeeks < 4) {
				return `${prefix}${diffWeeks} week${diffWeeks !== 1 ? 's' : ''}${suffix}`
			} else if (diffMonths < 12) {
				return `${prefix}${diffMonths} month${diffMonths !== 1 ? 's' : ''}${suffix}`
			} else {
				return `${prefix}${diffYears} year${diffYears !== 1 ? 's' : ''}${suffix}`
			}
		} catch (e) {
			return 'Invalid date'
		}
	}

	function formatAbsoluteDate(dateString) {
		if (!dateString) {
			return ''
		}

		try {
			const date = new Date(dateString)
			if (isNaN(date.getTime())) {
				return ''
			}
			return date.toLocaleString()
		} catch (e) {
			return ''
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
	.action-item {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 1rem 0;
		border-bottom: 1px solid #e0e0e0;
	}

	.action-item:last-child {
		border-bottom: none;
	}

	.action-description {
		display: flex;
		align-items: flex-start;
		gap: 1rem;
		flex: 1;
	}

	.action-description h5 {
		color: #333;
	}

	.action-description p {
		color: #666;
	}

	dd {
		display: flex;
		align-items: center;
		gap: 0.5rem;
	}

	.profile-link-button {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		padding: 0.25rem 0.5rem;
		border: 1px solid #ddd;
		border-radius: 0.25rem;
		background-color: #f5f5f5;
		color: #333;
		cursor: pointer;
		transition: all 0.2s ease;
		font-size: 0.875rem;
	}

	.profile-link-button:hover {
		background-color: #e0e0e0;
		border-color: #bbb;
	}

	.profile-link-button:active {
		background-color: #d0d0d0;
	}
</style>
