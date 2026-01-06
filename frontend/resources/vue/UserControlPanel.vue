<template>
	<Section
		title="User Control Panel"
		subtitle="Manage your account settings and preferences"
		classes="user-control-panel"
	>
		<template #toolbar>
			<button @click="refreshUserData" :disabled="!clientReady" class="neutral">
				<Icon icon="material-symbols:refresh" />
			</button>
		</template>

		<div v-if="errorMessage">
			<p class="inline-notification error">{{ errorMessage }}</p>
		</div>

		<div v-if="loading" class="icon-and-text">
			<Icon icon="eos-icons:loading" width="24" height="24" />
			<span>Loading user data...</span>
		</div>

		<div v-else-if="userData">
			<!-- User Profile Card -->
			<div class="icon-and-text" style="margin-bottom: 2em;">
				<Icon icon="mdi:account-circle" width="64" height="64" />
				<div>
					<h3 style="margin: 0 0 0.5em 0;">{{ userData.username }}</h3>
					<p style="margin: 0 0 0.25em 0;">{{ userData.email || 'No email provided' }}</p>
					<p style="margin: 0; font-size: 0.9em;">{{ userData.role || 'User' }}</p>
				</div>
			</div>

			<!-- Account Information -->
			<dl class="account-info">
				<dt>Username</dt>
				<dd>{{ userData.username }}</dd>

				<dt>Email</dt>
				<dd>{{ userData.email || 'Not provided' }}</dd>

				<dt>Role</dt>
				<dd>{{ userData.role || 'User' }}</dd>

				<dt>Account Created</dt>
				<dd>{{ formatDate(userData.createdAt) }}</dd>

				<dt>Last Login</dt>
				<dd>{{ formatDate(userData.lastLoginAt) }}</dd>
			</dl>
		</div>
	</Section>

	<Section
		v-if="!loading && userData"
		title="Quick Actions"
		subtitle="Quick navigation to common tasks"
	>
		<Navigation ref="localNavigation">
			<NavigationGrid />
		</Navigation>
	</Section>

	<Section
		v-if="!loading && userData"
		title="Session Management"
		subtitle="Manage your current session"
	>
		<div class="icon-and-text" style="align-items: center; justify-content: space-between;">
			<div class="icon-and-text">
				<Icon icon="mdi:logout" width="24" height="24" />
				<div>
					<h5 style="margin: 0 0 0.25em 0;">Sign Out</h5>
					<p style="margin: 0; font-size: 0.9em;">End your current session and return to the login page.</p>
				</div>
			</div>
			<button @click="logout" :disabled="loggingOut" class="bad">
				<Icon v-if="loggingOut" icon="eos-icons:loading" width="16" height="16" />
				<Icon v-else icon="mdi:logout" width="16" height="16" />
				<span>{{ loggingOut ? 'Signing Out...' : 'Sign Out' }}</span>
			</button>
		</div>
	</Section>
</template>

<script setup>
	import { ref, onMounted } from 'vue';
	import { useRouter } from 'vue-router';
	import { waitForClient } from '../javascript/util';
	import { Icon } from '@iconify/vue';
	import Section from 'picocrank/vue/components/Section.vue';
	import Navigation from 'picocrank/vue/components/Navigation.vue';
	import NavigationGrid from 'picocrank/vue/components/NavigationGrid.vue';
	import { UserMultiple03Icon, SettingsIcon, KeyIcon, SecurityValidationIcon, ActivityIcon } from '@hugeicons/core-free-icons';

	const router = useRouter();
	
	const clientReady = ref(false);
	const loading = ref(true);
	const errorMessage = ref('');
	const userData = ref(null);
	const localNavigation = ref(null);

	// Logout state
	const loggingOut = ref(false);

	async function refreshUserData() {
		loading.value = true;
		errorMessage.value = '';
		
		try {
			await waitForClient();
			
			// Get current user status
			const status = await window.client.getStatus();
			userData.value = {
				username: status.username,
				email: status.email || null,
				role: status.role || 'User',
				createdAt: status.createdAt || null,
				lastLoginAt: status.lastLoginAt || null
			};

		} catch (error) {
			console.error('Error fetching user data:', error);
			errorMessage.value = 'Failed to load user data: ' + error.message;
		} finally {
			loading.value = false;
		}
	}

	function formatDate(dateString) {
		if (!dateString) return 'Unknown';
		try {
			return new Date(dateString).toLocaleDateString();
		} catch {
			return 'Invalid date';
		}
	}

	function goToSocialAccounts() {
		router.push('/social-accounts');
	}

	function goToSettings() {
		router.push('/settings');
	}

	function goToApiKeys() {
		router.push('/api-keys');
	}

	function goToChangePassword() {
		router.push('/change-password');
	}

	function goToBrowserDiagnostics() {
		router.push('/browser-diagnostics');
	}

	async function logout() {
		if (loggingOut.value) return;

		loggingOut.value = true;

		try {
			await waitForClient();
			
			const response = await window.client.logout({});

			if (response.standardResponse.success) {
				// Clear the global login state
				window.isLoggedIn = false;
				
				// Redirect to the login page
				router.push('/');
				
				// Reload the page to ensure clean state
				window.location.reload();
			} else {
				console.error('Logout failed:', response.standardResponse.message);
				// Even if logout fails, redirect to login page
				window.isLoggedIn = false;
				router.push('/');
				window.location.reload();
			}
		} catch (error) {
			console.error('Error during logout:', error);
			// Even if logout fails, redirect to login page
			window.isLoggedIn = false;
			router.push('/');
			window.location.reload();
		} finally {
			loggingOut.value = false;
		}
	}

	onMounted(async () => {
		await waitForClient();
		clientReady.value = true;
		await refreshUserData();

		// Setup quick actions navigation
		if (localNavigation.value) {
			localNavigation.value.addCallback('Manage Social Accounts', goToSocialAccounts, {
				icon: UserMultiple03Icon,
				name: 'social-accounts',
				description: 'Manage your connected social media accounts'
			});

			localNavigation.value.addCallback('Account Settings', goToSettings, {
				icon: SettingsIcon,
				name: 'account-settings',
				description: 'Manage your account configuration'
			});

			localNavigation.value.addCallback('API Keys', goToApiKeys, {
				icon: KeyIcon,
				name: 'api-keys',
				description: 'View and manage your API keys'
			});

			localNavigation.value.addCallback('Change Password', goToChangePassword, {
				icon: SecurityValidationIcon,
				name: 'change-password',
				description: 'Change your account password'
			});

			localNavigation.value.addCallback('Browser Diagnostics', goToBrowserDiagnostics, {
				icon: ActivityIcon,
				name: 'browser-diagnostics',
				description: 'View browser and system information for troubleshooting'
			});
		}
	});
</script>

<style scoped>
	.account-info {
		display: grid;
		grid-template-columns: 200px 1fr;
		column-gap: 1em;
		row-gap: 0.25em;
		margin: 0;
	}

	.account-info dt {
		font-weight: bold;
	}

	.account-info dd {
		margin: 0;
	}
</style>
