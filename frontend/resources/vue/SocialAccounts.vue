<template>
	<Section
		title="Social Accounts"
		subtitle="This page shows a list of social accounts that you have configured."
		classes="social-accounts"
		:padding="false"
	>
		<template #toolbar>
			<button
				type="button"
				class="neutral"
				:title="viewToggleTitle"
				:disabled="!clientReady"
				@click="cycleViewMode"
			>
				<Icon :icon="viewToggleIcon" />
			</button>
			<router-link :to="{ name: 'addSocialAccount' }" class="button good" title="Add social account">
				<Icon icon="material-symbols:add" />
				Add account
			</router-link>
			<button type="button" @click="refreshAccounts()" :disabled="!clientReady" class="neutral" title="Refresh">
				<Icon icon="material-symbols:refresh" />
			</button>
		</template>

		<div v-if="errorMessage" class="padding">
			<p class="inline-notification error">{{ errorMessage }}</p>
		</div>
		<div v-else>
			<div v-if="accounts.length === 0" class="padding">
				<p class="inline-notification note">No social accounts connected yet.</p>
			</div>
			<table v-else-if="viewMode === 'table'" class="data-table">
				<thead>
					<tr>
						<th>Identity</th>
						<th>Owner</th>
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
						<td>
							<span v-if="account.isOwner" class="owner-tag self">you</span>
							<span v-else-if="account.ownerUsername" class="owner-tag">{{ account.ownerUsername }}</span>
							<span v-else class="owner-tag none">—</span>
						</td>
						<td align="right">
							<button type="button" @click="openProfile(account)" class="neutral" :title="'Open ' + account.identity + ' profile'">
								<Icon icon="material-symbols:open-in-new" />
							</button>
							<template v-if="account.canManage">
								&nbsp;
								<button type="button" @click="refreshAccount(account.id)" class="good" :disabled="isAccountRefreshing(account.id)">
									<Icon v-if="isAccountSuccess(account.id)" icon="material-symbols:check-circle" />
									<Icon v-else-if="isAccountRefreshing(account.id)" icon="material-symbols:hourglass-top" />
									<Icon v-else icon="material-symbols:refresh" />
								</button>
							</template>
						</td>
					</tr>
				</tbody>
			</table>
			<div v-else class="padding accounts-grid-wrap">
				<Navigation ref="accountsNavigation">
					<NavigationGrid />
				</Navigation>
			</div>
		</div>
	</Section>
</template>

<script setup>
	import { Icon } from '@iconify/vue';
	import { ref, computed, onMounted, inject, onActivated, watch, nextTick } from 'vue';
	import { useRouter } from 'vue-router';
	import { waitForClient, connectorHugeIcon } from '../javascript/util';
	import Section from 'picocrank/vue/components/Section.vue';
	import Navigation from 'picocrank/vue/components/Navigation.vue';
	import NavigationGrid from 'picocrank/vue/components/NavigationGrid.vue';

	const VIEW_MODES = ['table', 'grid'];
	const VIEW_STORAGE_KEY = 'socialAccounts.viewMode';

	const router = useRouter();
	const clientReady = ref(false);
	const errorMessage = ref('');
	const accounts = ref([]);
	const showErrorDialog = inject('showSectionError');
	const refreshingAccounts = ref(new Set());
	const successAccounts = ref(new Set());
	const accountsNavigation = ref(null);

	function loadViewMode() {
		const saved = localStorage.getItem(VIEW_STORAGE_KEY);
		return VIEW_MODES.includes(saved) ? saved : 'table';
	}

	const viewMode = ref(loadViewMode());

	const viewToggleIcon = computed(() =>
		viewMode.value === 'table' ? 'material-symbols:grid-view' : 'material-symbols:table-rows'
	);

	const viewToggleTitle = computed(() =>
		viewMode.value === 'table' ? 'Switch to grid view' : 'Switch to table view'
	);

	function cycleViewMode() {
		const index = VIEW_MODES.indexOf(viewMode.value);
		viewMode.value = VIEW_MODES[(index + 1) % VIEW_MODES.length];
		localStorage.setItem(VIEW_STORAGE_KEY, viewMode.value);
	}

	function isAccountRefreshing(accountId) {
		return refreshingAccounts.value.has(accountId);
	}

	function isAccountSuccess(accountId) {
		return successAccounts.value.has(accountId);
	}

	function ownerLabel(account) {
		if (account.isOwner) {
			return 'Owner: you';
		}
		if (account.ownerUsername) {
			return `Owner: ${account.ownerUsername}`;
		}
		return 'Owner: —';
	}

	function accountDescription(account) {
		const parts = [ownerLabel(account)];
		if (account.connector) {
			parts.push(account.connector);
		}
		if (account.canManage) {
			parts.push('You can manage this account');
		}
		return parts.join(' · ');
	}

	function updateAccountsNavigation() {
		const nav = accountsNavigation.value;
		if (!nav || viewMode.value !== 'grid') {
			return;
		}

		nav.clearNavigationLinks();

		for (const account of accounts.value) {
			nav.addCallback(account.identity, () => {
				router.push({ name: 'socialAccountDetails', params: { id: account.id } });
			}, {
				icon: connectorHugeIcon(account.connector),
				name: `social-account-${account.id}`,
				description: accountDescription(account),
			});
		}
	}

	function getProfileUrl(account) {
		if (!account || !account.identity) {
			return null;
		}

		const identity = account.identity;
		const connector = account.connector;

		switch (connector) {
			case 'bluesky':
				return `https://bsky.app/profile/${identity}`;
			case 'x':
				return `https://x.com/${identity}`;
			case 'mastodon': {
				const homeserver = 'https://mastodon.social';
				return `${homeserver}/@${identity}`;
			}
			default:
				return null;
		}
	}

	function openProfile(account) {
		const profileUrl = getProfileUrl(account);
		if (profileUrl) {
			window.open(profileUrl, '_blank', 'noopener,noreferrer');
		} else {
			showErrorDialog?.('Unable to determine profile URL for this account type.');
		}
	}

	function refreshAccount(accountId) {
		refreshingAccounts.value.add(accountId);

		window.client.refreshSocialAccount({ id: accountId })
			.then((ret) => {
				if (!ret.standardResponse.success) {
					const message = ret.standardResponse.message.toLowerCase();
					if (message.includes('re-authentication required') || message.includes('reauthentication required')) {
						const account = accounts.value.find((a) => a.id === accountId);
						if (account && account.connector) {
							window.client.startOAuth({ connectorId: account.connector })
								.then((oauthRes) => {
									window.location.href = oauthRes.url;
								})
								.catch((oauthError) => {
									console.error('Error starting OAuth flow:', oauthError);
									showErrorDialog?.('Failed to start re-authentication. Please try connecting the account again from the OAuth Services section.');
									refreshingAccounts.value.delete(accountId);
								});
							return;
						}
						showErrorDialog?.('Re-authentication required, but could not determine connector. Please reconnect the account from the OAuth Services section.');
					} else {
						showErrorDialog?.(ret.standardResponse.message);
					}
				}

				if (ret.standardResponse.success) {
					successAccounts.value.add(accountId);
					setTimeout(() => {
						successAccounts.value.delete(accountId);
					}, 1500);
				}

				refreshAccounts();
			})
			.catch((error) => {
				errorMessage.value = 'Failed to refresh social account: ' + error.message;
				console.error('Error refreshing social account:', error);
			})
			.finally(() => {
				refreshingAccounts.value.delete(accountId);
			});
	}

	async function refreshAccounts() {
		return await window.client.getSocialAccounts()
			.then((ret) => {
				ret.accounts.sort((a, b) => Number(a.id) - Number(b.id));
				accounts.value = ret.accounts || [];
			})
			.catch((error) => {
				errorMessage.value = 'Failed to fetch social accounts: ' + error.message;
				console.error('Error fetching social accounts:', error);
				return [];
			});
	}

	watch([accounts, viewMode, accountsNavigation], async () => {
		if (viewMode.value === 'grid' && accounts.value.length > 0) {
			await nextTick();
			updateAccountsNavigation();
		}
	}, { deep: true });

	onMounted(async () => {
		await waitForClient();
		clientReady.value = true;
		await refreshAccounts();
	});

	onActivated(() => {
		refreshAccounts();
	});
</script>

<style scoped>
	.owner-tag {
		font-size: 0.85em;
		opacity: 0.8;
	}

	.owner-tag.self {
		font-weight: 600;
	}

	.owner-tag.none {
		opacity: 0.4;
	}

	.accounts-grid-wrap {
		padding-top: 0.25em;
	}
</style>
