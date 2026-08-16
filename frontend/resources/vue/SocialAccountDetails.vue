<template>
	<Section
		:title="account?.identity || 'Social Account'"
		:subtitle="account ? account.connector : 'Loading account details'"
		classes="social-account-details"
		:padding="false"
	>
		<template #toolbar>
			<button
				v-if="getProfileUrl(account)"
				type="button"
				class="inline-icon neutral"
				aria-label="Open profile"
				:disabled="!clientReady || loading"
				title="Open Profile"
				@click="openProfile"
			>
				<HugeiconsIcon
					:icon="LinkSquare01Icon"
					width="1em"
					height="1em"
					:strokeWidth="iconStrokeWidth"
					aria-hidden="true"
				/>
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
					type="button"
					class="inline-icon neutral small"
					:aria-label="'Open ' + account.identity + ' profile'"
					:title="'Open ' + account.identity + ' profile'"
					@click="openProfile"
				>
					<HugeiconsIcon
						:icon="LinkSquare01Icon"
						width="1em"
						height="1em"
						:strokeWidth="iconStrokeWidth"
						aria-hidden="true"
					/>
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
			<button type="button" class="inline-icon good" :disabled="!clientReady || loading" @click="refreshAccount">
				<HugeiconsIcon
					:icon="RefreshIcon"
					width="1em"
					height="1em"
					:strokeWidth="iconStrokeWidth"
					aria-hidden="true"
				/>
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
			<button
				type="button"
				class="inline-icon"
				:class="account?.active ? 'warning' : 'good'"
				:disabled="!clientReady || loading"
				@click="toggleActive"
			>
				<HugeiconsIcon
					:icon="account?.active ? ToggleOffIcon : ToggleOnIcon"
					width="1em"
					height="1em"
					:strokeWidth="iconStrokeWidth"
					aria-hidden="true"
				/>
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
			<button type="button" class="inline-icon bad" :disabled="!clientReady || loading" @click="deleteAccount">
				<HugeiconsIcon
					:icon="Delete02Icon"
					width="1em"
					height="1em"
					:strokeWidth="iconStrokeWidth"
					aria-hidden="true"
				/>
				<span>Delete</span>
			</button>
		</div>
	</Section>

	<Section
		v-if="!loading && account"
		title="Recent logs"
		subtitle="The 5 most recent application events for this social account"
		classes="social-account-logs"
		:padding="false"
	>
		<template #toolbar>
			<button
				type="button"
				class="inline-icon neutral"
				aria-label="Refresh logs"
				:disabled="logsLoading"
				title="Refresh logs"
				@click="loadAccountLogs"
			>
				<HugeiconsIcon
					:icon="RefreshIcon"
					width="1em"
					height="1em"
					:strokeWidth="iconStrokeWidth"
					aria-hidden="true"
				/>
			</button>
		</template>

		<div v-if="logsLoading" style="padding: 1rem;">
			<p>Loading logs...</p>
		</div>
		<div v-else-if="logsError" style="padding: 1rem;">
			<p class="inline-notification error">{{ logsError }}</p>
		</div>
		<div v-else-if="accountLogs.length === 0" style="padding: 1rem;">
			<p class="inline-notification note">No logs found for this account.</p>
		</div>
		<table v-else class="data-table">
			<thead>
				<tr>
					<th>Time</th>
					<th>Level</th>
					<th>Message</th>
				</tr>
			</thead>
			<tbody>
				<tr v-for="entry in accountLogs" :key="entry.id" :class="getLogLevelClass(entry.level)">
					<td>{{ formatLogDate(entry.createdAt) }}</td>
					<td>
						<span class="log-level">{{ entry.level }}</span>
					</td>
					<td>{{ entry.message }}</td>
				</tr>
			</tbody>
		</table>
	</Section>

	<Section
		v-if="!loading && account && canManageSharing"
		subtitle="Grant read, post, or manage access to user groups; members of those groups inherit the permissions"
		classes="social-account-sharing"
		:padding="shares.length === 0"
	>
		<template #title>
			<span class="section-title-with-icon">
				<HugeiconsIcon :icon="UserGroupIcon" width="22" height="22" aria-hidden="true" />
				Sharing
			</span>
		</template>

		<template #toolbar>
			<button
				type="button"
				class="inline-icon neutral"
				aria-label="Refresh"
				title="Refresh"
				:disabled="sharesLoading"
				@click="loadShares"
			>
				<HugeiconsIcon
					:icon="RefreshIcon"
					width="1em"
					height="1em"
					:strokeWidth="iconStrokeWidth"
					aria-hidden="true"
				/>
			</button>
			<button
				type="button"
				class="inline-icon neutral"
				aria-label="Save shares"
				title="Save shares"
				:disabled="sharesLoading || sharesSaving"
				@click="saveShares"
			>
				<HugeiconsIcon
					:icon="SaveIcon"
					width="1em"
					height="1em"
					:strokeWidth="iconStrokeWidth"
					aria-hidden="true"
				/>
			</button>
			<button
				type="button"
				class="inline-icon good"
				aria-label="Add user group"
				title="Add user group"
				:disabled="sharesLoading"
				@click="openShareGroupLookup"
			>
				<HugeiconsIcon
					:icon="Add01Icon"
					width="1em"
					height="1em"
					:strokeWidth="iconStrokeWidth"
					aria-hidden="true"
				/>
			</button>
		</template>

		<div
			v-if="sharesError"
			class="inline-notification error"
			:class="{ 'list-banner-pad': shares.length > 0 }"
		>{{ sharesError }}</div>
		<div v-if="sharesLoading && !shares.length" class="muted">Loading shares…</div>

		<template v-else>
			<p v-if="!shares.length" class="inline-notification note">Not shared with any user groups.</p>

			<Table
				v-else
				class="shares-table-wrap"
				:data="shares"
				:headers="shareTableHeaders"
			>
				<template #cell-groupName="{ value }">
					<strong>{{ value }}</strong>
				</template>
				<template #cell-canRead="{ row }">
					<div class="share-check-cell">
						<input type="checkbox" v-model="row.canRead" :aria-label="`Read access for ${row.groupName}`" />
					</div>
				</template>
				<template #cell-canPost="{ row }">
					<div class="share-check-cell">
						<input type="checkbox" v-model="row.canPost" :aria-label="`Post access for ${row.groupName}`" />
					</div>
				</template>
				<template #cell-canManage="{ row }">
					<div class="share-check-cell">
						<input type="checkbox" v-model="row.canManage" :aria-label="`Manage access for ${row.groupName}`" />
					</div>
				</template>
				<template #cell-actions="{ row }">
					<div class="actions-cell">
						<button type="button" class="bad small" @click="removeShare(row)">Remove</button>
					</div>
				</template>
			</Table>

			<div v-if="sharesMessage" :class="{ 'list-banner-pad': shares.length > 0 }">
				<p :class="'inline-notification ' + sharesMessageType">{{ sharesMessage }}</p>
			</div>
		</template>
	</Section>

	<UserGroupLookupDialog
		ref="shareGroupLookup"
		title="Share with user groups"
		subtitle="Select groups to grant access. Groups already shared are hidden."
		multiple
		confirm-label="Add groups"
		:exclude-group-ids="sharedGroupIds"
		@picked="onShareGroupsPicked"
	/>
</template>

<script setup>
	import { ref, computed, onMounted } from 'vue';
	import { useRoute, useRouter } from 'vue-router';
	import { Icon } from '@iconify/vue';
	import { HugeiconsIcon } from '@hugeicons/vue';
	import {
		Add01Icon,
		Delete02Icon,
		LinkSquare01Icon,
		RefreshIcon,
		SaveIcon,
		ToggleOffIcon,
		ToggleOnIcon,
		UserGroupIcon,
	} from '@hugeicons/core-free-icons';
	import Section from 'picocrank/vue/components/Section.vue';
	import Table from 'picocrank/vue/components/Table.vue';
	import UserGroupLookupDialog from './UserGroupLookupDialog.vue';

	const route = useRoute();
	const router = useRouter();
	const iconStrokeWidth = 2.5;
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
	const shareGroupLookup = ref(null)

	const accountLogs = ref([])
	const logsLoading = ref(false)
	const logsError = ref('')

	const canManageSharing = computed(() => {
		if (!account.value) return false
		if (account.value.isOwner) return true
		if (statusSuper.value) return true
		if (Array.isArray(statusPerms.value) && statusPerms.value.includes('social-accounts.view-all')) return true
		return false
	})

	const sharedGroupIds = computed(() => shares.value.map((s) => s.userGroupId))

	const shareTableHeaders = [
		{ key: 'groupName', label: 'User group', sortable: true },
		{ key: 'canRead', label: 'Read', sortable: false, width: '6rem' },
		{ key: 'canPost', label: 'Post', sortable: false, width: '6rem' },
		{ key: 'canManage', label: 'Manage', sortable: false, width: '6rem' },
		{ key: 'actions', label: 'Actions', sortable: false, width: '6rem' },
	]

	function openShareGroupLookup() {
		shareGroupLookup.value?.open()
	}

	function onShareGroupsPicked(groups) {
		const existing = new Set(sharedGroupIds.value)
		for (const group of groups) {
			if (existing.has(group.id)) continue
			shares.value.push({
				userGroupId: group.id,
				groupName: group.name,
				canRead: true,
				canPost: false,
				canManage: false,
			})
			existing.add(group.id)
		}
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
				await Promise.all([fetchLastPosted(), loadAccountLogs()])
			}
		} catch (e) {
			error.value = `Failed to load account: ${e.message || e}`
		} finally {
			loading.value = false
		}
	}

	async function loadAccountLogs() {
		if (!accountId.value) return
		logsLoading.value = true
		logsError.value = ''
		try {
			const res = await window.client.getLogs({
				limit: 5,
				relatedSocialAccountId: accountId.value,
			})
			accountLogs.value = res.logs || []
		} catch (e) {
			logsError.value = `Failed to load logs: ${e.message || e}`
			accountLogs.value = []
		} finally {
			logsLoading.value = false
		}
	}

	function formatLogDate(dateString) {
		if (!dateString) return '—'
		try {
			const date = new Date(dateString)
			if (Number.isNaN(date.getTime())) return dateString
			return date.toLocaleString()
		} catch {
			return dateString
		}
	}

	function getLogLevelClass(level) {
		const levelLower = (level || '').toLowerCase()
		if (levelLower.includes('error')) return 'log-error'
		if (levelLower.includes('warn')) return 'log-warn'
		if (levelLower.includes('info')) return 'log-info'
		if (levelLower.includes('debug')) return 'log-debug'
		return ''
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

	function removeShare(row) {
		const idx = shares.value.findIndex((s) => s.userGroupId === row.userGroupId)
		if (idx >= 0) shares.value.splice(idx, 1)
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
		await loadShares()
	})
</script>

<style scoped>
	.action-item {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 1rem 0;
		border-bottom: 1px solid var(--border-color, #e0e0e0);
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
		color: var(--text-primary, #333);
	}

	.action-description p {
		color: var(--text-secondary, #666);
	}

	html[data-theme="dark"] {
		.action-description h5 {
			color: #ddd;
		}

		.action-description p {
			color: #aaa;
		}
	}

	dd {
		display: flex;
		align-items: center;
		gap: 0.5rem;
	}

	.share-check-cell {
		text-align: center;
	}

	.section-title-with-icon {
		display: inline-flex;
		align-items: center;
		gap: 0.45em;
		vertical-align: middle;
	}

	.list-banner-pad {
		padding-left: 1em;
		padding-right: 1em;
	}

	.shares-table-wrap {
		margin-top: 0.5rem;
		margin-bottom: 0.5rem;
	}

	.actions-cell {
		text-align: right;
	}
</style>
