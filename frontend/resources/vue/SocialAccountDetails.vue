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
			<dt>Owner</dt>
			<dd>
				<span v-if="account.isOwner">{{ account.ownerUsername }} (you)</span>
				<span v-else-if="account.ownerUsername">{{ account.ownerUsername }}</span>
				<span v-else>—</span>
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
		v-if="!loading && account && account.canManage"
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

	<Section
		v-if="!loading && account && canManageSharing"
		title="Sharing"
		subtitle="Grant read, post, or manage access to user groups; members of those groups inherit the permissions"
		:padding="false"
	>
		<div v-if="sharesLoading" style="padding: 1rem;">
			<p>Loading shares...</p>
		</div>
		<div v-else-if="sharesError" style="padding: 1rem;">
			<p class="inline-notification error">{{ sharesError }}</p>
		</div>
		<div v-else>
			<table class="data-table" v-if="shares.length > 0">
				<thead>
					<tr>
						<th>User group</th>
						<th class="share-check-col">Read</th>
						<th class="share-check-col">Post</th>
						<th class="share-check-col">Manage</th>
						<th class="share-check-col"></th>
					</tr>
				</thead>
				<tbody>
					<tr v-for="(share, idx) in shares" :key="share.userGroupId">
						<td>{{ share.groupName }}</td>
						<td class="share-check-col"><input type="checkbox" v-model="shares[idx].canRead" /></td>
						<td class="share-check-col"><input type="checkbox" v-model="shares[idx].canPost" /></td>
						<td class="share-check-col"><input type="checkbox" v-model="shares[idx].canManage" /></td>
						<td class="share-check-col">
							<button class="bad small" @click="removeShare(idx)" title="Remove share">
								<Icon icon="material-symbols:close" width="14" height="14" />
							</button>
						</td>
					</tr>
				</tbody>
			</table>
			<p v-else class="inline-notification note">Not shared with any user groups.</p>

			<div class="share-add-row">
				<select v-model="addShareGroupId" class="share-user-select">
					<option value="">Add user group...</option>
					<option v-for="g in availableGroups" :key="g.id" :value="g.id">{{ g.name }}</option>
				</select>
				<button type="button" class="good" :disabled="!addShareGroupId" @click="addShare">
					<Icon icon="material-symbols:group-add" width="16" height="16" />
					Add
				</button>
				<span style="flex:1"></span>
				<button class="good" :disabled="sharesSaving" @click="saveShares">
					<Icon icon="material-symbols:save" width="16" height="16" />
					{{ sharesSaving ? 'Saving...' : 'Save' }}
				</button>
			</div>
			<div v-if="sharesMessage" style="padding: 0 1rem 1rem 1rem;">
				<p :class="'inline-notification ' + sharesMessageType">{{ sharesMessage }}</p>
			</div>
		</div>
	</Section>
</template>

<script setup>
	import { ref, computed, onMounted } from 'vue';
	import { useRoute, useRouter } from 'vue-router';
	import { Icon } from '@iconify/vue';
	import Section from 'picocrank/vue/components/Section.vue';

	const route = useRoute();
	const router = useRouter();
	const clientReady = ref(false)
	const loading = ref(true)
	const error = ref('')
	const accountId = ref(0)
	const account = ref(null)
	const lastPosted = ref(null)

	const statusPerms = ref([])
	const statusSuper = ref(false)

	const shares = ref([])
	const sharesLoading = ref(false)
	const sharesError = ref('')
	const sharesSaving = ref(false)
	const sharesMessage = ref('')
	const sharesMessageType = ref('good')
	const allGroups = ref([])
	const addShareGroupId = ref('')

	const canManageSharing = computed(() => {
		if (!account.value) return false
		if (account.value.isOwner) return true
		if (statusSuper.value) return true
		if (Array.isArray(statusPerms.value) && statusPerms.value.includes('social-accounts.view-all')) return true
		return false
	})

	const availableGroups = computed(() => {
		const sharedIds = new Set(shares.value.map((s) => s.userGroupId))
		return allGroups.value.filter((g) => !sharedIds.has(g.id))
	})

	async function loadAllGroups() {
		try {
			const res = await window.client.listUserGroups({})
			allGroups.value = res.groups || []
		} catch (e) {
			console.error('Failed to load user groups for sharing:', e)
		}
	}

	function addShare() {
		const gid = Number(addShareGroupId.value)
		if (!gid) return
		const group = allGroups.value.find((g) => g.id === gid)
		if (!group) return
		shares.value.push({
			userGroupId: gid,
			groupName: group.name,
			canRead: true,
			canPost: false,
			canManage: false,
		})
		addShareGroupId.value = ''
	}

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

	async function loadShares() {
		if (!canManageSharing.value) return
		sharesLoading.value = true
		sharesError.value = ''
		try {
			const res = await window.client.getSocialAccountShares({ socialAccountId: accountId.value })
			shares.value = (res.shares || []).map(s => ({
				userGroupId: s.userGroupId,
				groupName: s.groupName,
				canRead: s.canRead,
				canPost: s.canPost,
				canManage: s.canManage,
			}))
		} catch (e) {
			sharesError.value = `Failed to load shares: ${e.message || e}`
		} finally {
			sharesLoading.value = false
		}
	}

	function removeShare(idx) {
		shares.value.splice(idx, 1)
	}

	async function saveShares() {
		sharesSaving.value = true
		sharesMessage.value = ''
		try {
			await window.client.setSocialAccountShares({
				socialAccountId: accountId.value,
				shares: shares.value.map(s => ({
					userGroupId: s.userGroupId,
					groupName: s.groupName,
					canRead: s.canRead,
					canPost: s.canPost,
					canManage: s.canManage,
				})),
			})
			sharesMessage.value = 'Shares saved.'
			sharesMessageType.value = 'good'
			await loadShares()
		} catch (e) {
			sharesMessage.value = `Failed to save shares: ${e.message || e}`
			sharesMessageType.value = 'error'
		} finally {
			sharesSaving.value = false
		}
	}

	onMounted(async () => {
		accountId.value = parseInt(route.params.id, 10) || 0
		await waitForClient()
		clientReady.value = true

		const st = await window.client.getStatus({})
		statusPerms.value = st.rbacPermissions || []
		statusSuper.value = Boolean(st.rbacIsSuperuser)

		await fetchAccount()
		await Promise.all([loadShares(), loadAllGroups()])
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

	.share-check-col {
		text-align: center;
		width: 5rem;
	}

	.share-add-row {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		padding: 0.75rem 1rem;
	}

	button.small {
		padding: 0.2rem 0.4rem;
		min-width: auto;
	}

	.share-user-select {
		max-width: 16rem;
	}
</style>
