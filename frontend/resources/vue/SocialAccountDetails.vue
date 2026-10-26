<template>
	<Section
		:title="account?.identity || 'Social Account'"
		:subtitle="account ? account.connector : 'Loading account details'"
		:icon="UserMultiple03Icon"
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
		<NotificationBlock v-else-if="error" type="error" :message="error" />
		<NotificationBlock v-else-if="!account" type="note" message="Account not found." />
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

	<SocialAccountPostPerformance
		v-if="!loading && account"
		:social-account-id="account.id"
	/>

	<Section
		v-if="!loading && account && account.canManage"
		title="Account Actions"
		subtitle="Manage this social account"
		:icon="Settings01Icon"
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
	</Section>

	<Section
		v-if="!loading && account && account.canManage"
		title="Keyword Watch"
		subtitle="Monitor the network for keywords using this account's access. Matches emit the keyword.matched webhook. Posts older than the watch creation time are recorded without notifying."
		:icon="SearchList01Icon"
		classes="keyword-watches"
	>
		<dl v-if="watchCapabilities" class="watch-capabilities">
			<dt>Search watching</dt>
			<dd>{{ searchWatchDescription }}</dd>
			<dt>Live watching</dt>
			<dd>{{ liveWatchDescription }}</dd>
			<dt v-if="watchState?.acquisitionMode">Current mode</dt>
			<dd v-if="watchState?.acquisitionMode">{{ acquisitionModeDescription }}</dd>
			<dt v-if="watchState?.lastError">Last error</dt>
			<dd v-if="watchState?.lastError">{{ watchState.lastError }}</dd>
		</dl>

		<div class="keyword-add">
			<input
				v-model="newKeyword"
				type="text"
				placeholder="Add keyword or #hashtag"
				:disabled="watchesBusy"
				@keyup.enter="addKeywordWatch"
			/>
			<button type="button" class="inline-icon good" :disabled="watchesBusy || !newKeyword.trim()" @click="addKeywordWatch">
				<HugeiconsIcon :icon="Add01Icon" width="1em" height="1em" :strokeWidth="iconStrokeWidth" aria-hidden="true" />
				<span>Add</span>
			</button>
		</div>
		<NotificationBlock v-if="watchesError" type="error" :message="watchesError" />

		<Table
			class="keyword-watches-table-wrap"
			:data="watches"
			:headers="watchTableHeaders"
			:loading="watchesBusy && !watches.length"
		>
			<template #empty>
				No keywords configured.
			</template>
			<template #cell-keyword="{ row }">
				<input
					v-if="editingWatchId === row.id"
					v-model="editingWatchKeyword"
					type="text"
					class="keyword-edit-input"
					:disabled="watchesBusy"
					aria-label="Edit keyword"
					@keyup.enter="saveKeywordWatch(row)"
					@keyup.esc="cancelEditKeywordWatch"
				/>
				<span v-else>{{ row.keyword }}</span>
			</template>
			<template #cell-enabled="{ row }">
				<input
					type="checkbox"
					:checked="row.enabled"
					:disabled="watchesBusy || editingWatchId === row.id"
					:aria-label="`Enable watch for ${row.keyword}`"
					@change="toggleKeywordWatch(row, $event.target.checked)"
				/>
			</template>
			<template #cell-actions="{ row }">
				<div class="actions-cell">
					<template v-if="editingWatchId === row.id">
						<button type="button" class="good small" :disabled="watchesBusy || !editingWatchKeyword.trim()" @click="saveKeywordWatch(row)">
							Save
						</button>
						<button type="button" class="neutral small" :disabled="watchesBusy" @click="cancelEditKeywordWatch">
							Cancel
						</button>
					</template>
					<template v-else>
						<button type="button" class="neutral small" :disabled="watchesBusy" @click="beginEditKeywordWatch(row)">
							Edit
						</button>
						<button type="button" class="bad small" :disabled="watchesBusy" @click="removeKeywordWatch(row)">
							Remove
						</button>
					</template>
				</div>
			</template>
		</Table>

		<div v-if="recentMatches.length" class="recent-matches">
			<h5>Recent matches</h5>
			<table class="data-table">
				<thead>
					<tr>
						<th>When</th>
						<th>Keyword</th>
						<th>Author</th>
						<th>Mode</th>
						<th>Content</th>
					</tr>
				</thead>
				<tbody>
					<tr v-for="m in recentMatches" :key="m.id">
						<td>{{ formatLogDate(m.matchedAt) }}</td>
						<td>{{ m.keyword }}</td>
						<td>
							<a v-if="m.remoteUrl" :href="m.remoteUrl" target="_blank" rel="noopener">{{ m.authorHandle || m.authorId || 'post' }}</a>
							<span v-else>{{ m.authorHandle || m.authorId || '—' }}</span>
						</td>
						<td>{{ m.acquisitionMode }}</td>
						<td>{{ m.contentSnippet }}</td>
					</tr>
				</tbody>
			</table>
		</div>
	</Section>

	<Section
		v-if="!loading && account"
		title="Recent logs"
		subtitle="The 5 most recent application events for this social account"
		:icon="ActivityIcon"
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
		<NotificationBlock v-else-if="logsError" type="error" :message="logsError" />
		<NotificationBlock v-else-if="accountLogs.length === 0" type="note" message="No logs found for this account." />
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
		title="Sharing"
		subtitle="Grant read, post, or manage access to user groups; members of those groups inherit the permissions"
		:icon="UserGroupIcon"
		classes="social-account-sharing"
		:padding="false"
	>

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

		<NotificationBlock v-if="sharesError" type="error" :message="sharesError" />

		<Table
			class="shares-table-wrap"
			:data="shares"
			:headers="shareTableHeaders"
			:loading="sharesLoading"
		>
			<template #empty>
				Not shared with any user groups.
			</template>
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

		<NotificationBlock
			v-if="sharesMessage"
			:type="sharesMessageType"
			:message="sharesMessage"
		/>
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

	<DangerZone
		v-if="!loading && account && account.canManage"
		title="Danger zone"
		subtitle="Destructive actions for this social account"
		description="Show destructive actions"
		warning="Permanently remove this social account. This action cannot be undone."
	>
		<div role="toolbar" class="danger-zone-actions">
			<button type="button" class="inline-icon bad" :disabled="!clientReady || loading" @click="deleteAccount">
				<HugeiconsIcon
					:icon="Delete02Icon"
					width="1em"
					height="1em"
					:strokeWidth="iconStrokeWidth"
					aria-hidden="true"
				/>
				<span>Delete account</span>
			</button>
		</div>
	</DangerZone>
</template>

<script setup>
	import { ref, computed, onMounted } from 'vue';
	import { useRoute, useRouter } from 'vue-router';
	import { Icon } from '@iconify/vue';
	import { HugeiconsIcon } from '@hugeicons/vue';
	import {
		ActivityIcon,
		Add01Icon,
		Delete02Icon,
		LinkSquare01Icon,
		RefreshIcon,
		SaveIcon,
		SearchList01Icon,
		Settings01Icon,
		ToggleOffIcon,
		ToggleOnIcon,
		UserGroupIcon,
		UserMultiple03Icon,
	} from '@hugeicons/core-free-icons';
	import Section from 'picocrank/vue/components/Section.vue';
	import DangerZone from 'picocrank/vue/components/DangerZone.vue';
	import NotificationBlock from 'picocrank/vue/components/NotificationBlock.vue';
	import Table from 'picocrank/vue/components/Table.vue';
	import UserGroupLookupDialog from './UserGroupLookupDialog.vue';
	import SocialAccountPostPerformance from './SocialAccountPostPerformance.vue';

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

	const watches = ref([])
	const watchState = ref(null)
	const recentMatches = ref([])
	const watchCapabilities = ref(null)
	const watchesBusy = ref(false)
	const watchesError = ref('')
	const newKeyword = ref('')
	const editingWatchId = ref(0)
	const editingWatchKeyword = ref('')

	const watchTableHeaders = [
		{ key: 'keyword', label: 'Keyword', sortable: true },
		{ key: 'enabled', label: 'Enabled', sortable: false, width: '6rem' },
		{ key: 'actions', label: 'Actions', sortable: false, width: '10rem' },
	]

	const searchWatchDescription = computed(() => {
		if (!watchCapabilities.value) return ''
		if (watchCapabilities.value.supportsSearch) {
			return 'Supported. Japella periodically searches the network for your keywords.'
		}
		return 'Not supported on this network. Keyword matches will not be found via search polling.'
	})

	const liveWatchDescription = computed(() => {
		if (!watchCapabilities.value) return ''
		const fidelity = String(watchCapabilities.value.nativeFidelity || '')
		switch (fidelity) {
			case 'hashtag_only':
				return 'Supported for hashtags only (keywords that start with #). Those stream live; other keywords use search watching.'
			case 'exact_rule':
				return 'Supported. Keywords can be watched live with exact matching on this network.'
			case 'client_filter':
				return 'Supported with client-side filtering. Japella receives a live feed and matches your keywords locally.'
			case 'search_only':
				return 'Not available. This network does not offer a live keyword stream, so Japella relies on search watching.'
			default:
				if (watchCapabilities.value.supportsNative) {
					return 'Supported. Live keyword watching is available on this network.'
				}
				return 'Not available. This network does not offer a live keyword stream, so Japella relies on search watching when search is supported.'
		}
	})

	const acquisitionModeDescription = computed(() => {
		const mode = String(watchState.value?.acquisitionMode || '').toLowerCase()
		switch (mode) {
			case 'native':
				return 'Using live watching right now.'
			case 'search':
				return 'Using search watching right now.'
			case 'evaluating':
				return 'Choosing the best watching method for this account.'
			case 'degraded':
				return 'Temporarily limited (for example after rate limiting). Japella will retry when it can.'
			default:
				return watchState.value?.acquisitionMode || ''
		}
	})

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
				await Promise.all([fetchLastPosted(), loadAccountLogs(), loadKeywordWatches()])
			}
		} catch (e) {
			error.value = `Failed to load account: ${e.message || e}`
		} finally {
			loading.value = false
		}
	}

	async function loadKeywordWatches() {
		if (!accountId.value || !account.value?.canManage) return
		watchesError.value = ''
		try {
			const res = await window.client.listKeywordWatches({ socialAccountId: accountId.value })
			watches.value = res.watches || []
			watchState.value = res.state || null
			recentMatches.value = res.recentMatches || []
			watchCapabilities.value = {
				supportsSearch: !!res.supportsSearch,
				supportsNative: !!res.supportsNative,
				nativeFidelity: res.nativeFidelity || '',
			}
		} catch (e) {
			watchesError.value = `Failed to load keyword watches: ${e.message || e}`
			watches.value = []
			recentMatches.value = []
		}
	}

	async function addKeywordWatch() {
		const keyword = newKeyword.value.trim()
		if (!keyword || !accountId.value) return
		watchesBusy.value = true
		watchesError.value = ''
		try {
			await window.client.createKeywordWatch({
				socialAccountId: accountId.value,
				keyword,
				enabled: true,
			})
			newKeyword.value = ''
			await loadKeywordWatches()
		} catch (e) {
			watchesError.value = `Failed to add keyword: ${e.message || e}`
		} finally {
			watchesBusy.value = false
		}
	}

	async function toggleKeywordWatch(w, enabled) {
		watchesBusy.value = true
		watchesError.value = ''
		try {
			await window.client.updateKeywordWatch({
				id: w.id,
				keyword: w.keyword,
				enabled,
			})
			await loadKeywordWatches()
		} catch (e) {
			watchesError.value = `Failed to update keyword: ${e.message || e}`
			await loadKeywordWatches()
		} finally {
			watchesBusy.value = false
		}
	}

	function beginEditKeywordWatch(w) {
		editingWatchId.value = w.id
		editingWatchKeyword.value = w.keyword
		watchesError.value = ''
	}

	function cancelEditKeywordWatch() {
		editingWatchId.value = 0
		editingWatchKeyword.value = ''
	}

	async function saveKeywordWatch(w) {
		const keyword = editingWatchKeyword.value.trim()
		if (!keyword) return
		watchesBusy.value = true
		watchesError.value = ''
		try {
			await window.client.updateKeywordWatch({
				id: w.id,
				keyword,
				enabled: w.enabled,
			})
			cancelEditKeywordWatch()
			await loadKeywordWatches()
		} catch (e) {
			watchesError.value = `Failed to update keyword: ${e.message || e}`
		} finally {
			watchesBusy.value = false
		}
	}

	async function removeKeywordWatch(w) {
		if (!confirm(`Remove keyword watch "${w.keyword}"?`)) return
		watchesBusy.value = true
		watchesError.value = ''
		try {
			await window.client.deleteKeywordWatch({ id: w.id })
			await loadKeywordWatches()
		} catch (e) {
			watchesError.value = `Failed to remove keyword: ${e.message || e}`
		} finally {
			watchesBusy.value = false
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


	.shares-table-wrap {
		margin-top: 0.5rem;
		margin-bottom: 0.5rem;
	}

	.actions-cell {
		text-align: right;
	}

	.danger-zone-actions {
		display: flex;
		flex-wrap: wrap;
		gap: 0.5rem;
	}

	.keyword-add {
		display: flex;
		gap: 0.5rem;
		align-items: center;
		margin-bottom: 1rem;
	}

	.keyword-add input {
		flex: 1;
		max-width: 24rem;
	}

	.keyword-watches-table-wrap {
		margin-bottom: 0.5rem;
	}

	.keyword-edit-input {
		width: 100%;
		max-width: 24rem;
		box-sizing: border-box;
	}

	.watch-capabilities {
		display: grid;
		grid-template-columns: 9rem 1fr;
		gap: 0.45rem 1rem;
		margin: 0 0 1rem;
	}

	.watch-capabilities dt {
		font-weight: 600;
		opacity: 0.85;
	}

	.watch-capabilities dd {
		margin: 0;
	}

	.recent-matches {
		margin-top: 1.25rem;
	}
</style>
