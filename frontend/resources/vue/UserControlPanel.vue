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
			<div>
				<table class="data-table">
					<tbody>
						<tr>
							<td><strong>Username:</strong></td>
							<td>{{ userData.username }}</td>
						</tr>
						<tr>
							<td><strong>Email:</strong></td>
							<td>{{ userData.email || 'Not provided' }}</td>
						</tr>
						<tr>
							<td><strong>Role:</strong></td>
							<td>{{ userData.role || 'User' }}</td>
						</tr>
						<tr>
							<td><strong>Account Created:</strong></td>
							<td>{{ formatDate(userData.createdAt) }}</td>
						</tr>
						<tr>
							<td><strong>Last Login:</strong></td>
							<td>{{ formatDate(userData.lastLoginAt) }}</td>
						</tr>
					</tbody>
				</table>
			</div>
		</div>
	</Section>

	<Section
		v-if="!loading && userData"
		title="Connected Social Accounts"
		subtitle="Social media accounts linked to your profile"
	>
		<div v-if="socialAccounts.length === 0">
			<p class="inline-notification note">No social accounts connected.</p>
		</div>
		<table v-else class="data-table">
			<thead>
				<tr>
					<th>Platform</th>
					<th>Handle</th>
					<th>Status</th>
					<th>Connected</th>
				</tr>
			</thead>
			<tbody>
				<tr v-for="account in socialAccounts" :key="account.id">
					<td>
						<div class="icon-and-text">
							<Icon :icon="getSocialIcon(account.platform)" width="24" height="24" />
							<span>{{ account.platform }}</span>
						</div>
					</td>
					<td>{{ account.handle || account.identity || 'Unknown' }}</td>
					<td>
						<span :class="account.isActive ? 'good' : 'bad'">
							{{ account.isActive ? 'Active' : 'Inactive' }}
						</span>
					</td>
					<td>{{ formatDate(account.createdAt) }}</td>
				</tr>
			</tbody>
		</table>
	</Section>

	<Section
		v-if="!loading && userData"
		title="Change Password"
		subtitle="Update your account password"
	>
		<form @submit.prevent="changePassword">
			<div class="password-form-grid">
				<label for="current-password">Current Password:</label>
				<input 
					type="password" 
					id="current-password" 
					v-model="passwordForm.currentPassword" 
					required 
					:disabled="passwordChanging"
				/>
				
				<label for="new-password">New Password:</label>
				<div>
					<input 
						type="password" 
						id="new-password" 
						v-model="passwordForm.newPassword" 
						required 
						minlength="8"
						:disabled="passwordChanging"
					/>
					<small style="display: block; margin-top: 0.25em; font-size: 0.8em;">Password must be at least 8 characters long</small>
				</div>
				
				<label for="confirm-password">Confirm New Password:</label>
				<input 
					type="password" 
					id="confirm-password" 
					v-model="passwordForm.confirmPassword" 
					required 
					:disabled="passwordChanging"
				/>
				
				<div class="password-form-buttons">
					<button type="submit" :disabled="passwordChanging || !isPasswordFormValid" class="good">
						<Icon v-if="passwordChanging" icon="eos-icons:loading" width="16" height="16" />
						<Icon v-else icon="mdi:lock-reset" width="16" height="16" />
						<span>{{ passwordChanging ? 'Changing...' : 'Change Password' }}</span>
					</button>
					<button type="button" @click="resetPasswordForm" :disabled="passwordChanging" class="neutral">
						Reset
					</button>
				</div>
				
				<div v-if="passwordMessage" class="password-form-message">
					<div class="inline-notification" :class="passwordMessageType">
						{{ passwordMessage }}
					</div>
				</div>
			</div>
		</form>
	</Section>

	<Section
		v-if="!loading && userData"
		title="Quick Actions"
		subtitle="Quick navigation to common tasks"
	>
		<div style="display: flex; gap: 1em; flex-wrap: wrap;">
			<button @click="goToSocialAccounts" class="neutral">
				<Icon icon="mdi:account-multiple" />
				<span>Manage Social Accounts</span>
			</button>
			<button @click="goToSettings" class="neutral">
				<Icon icon="mdi:cog" />
				<span>Account Settings</span>
			</button>
			<button @click="goToApiKeys" class="neutral">
				<Icon icon="mdi:key" />
				<span>API Keys</span>
			</button>
		</div>
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
	import { ref, computed, onMounted } from 'vue';
	import { useRouter } from 'vue-router';
	import { waitForClient } from '../javascript/util';
	import { Icon } from '@iconify/vue';
	import Section from 'picocrank/vue/components/Section.vue';

	const router = useRouter();
	
	const clientReady = ref(false);
	const loading = ref(true);
	const errorMessage = ref('');
	const userData = ref(null);
	const socialAccounts = ref([]);

	// Password change form
	const passwordForm = ref({
		currentPassword: '',
		newPassword: '',
		confirmPassword: ''
	});
	const passwordChanging = ref(false);
	const passwordMessage = ref('');
	const passwordMessageType = ref('');

	// Logout state
	const loggingOut = ref(false);

	// Computed property for form validation
	const isPasswordFormValid = computed(() => {
		return passwordForm.value.currentPassword.length > 0 &&
			   passwordForm.value.newPassword.length >= 8 &&
			   passwordForm.value.confirmPassword.length > 0 &&
			   passwordForm.value.newPassword === passwordForm.value.confirmPassword;
	});

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

			// Get social accounts
			const socialAccountsResponse = await window.client.getSocialAccounts();
			socialAccounts.value = socialAccountsResponse.socialAccounts || [];
			
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

	function getSocialIcon(platform) {
		const iconMap = {
			'bluesky': 'bi:bluesky',
			'mastodon': 'mdi:mastodon',
			'x': 'mdi:twitter',
			'twitter': 'mdi:twitter',
			'facebook': 'mdi:facebook',
			'instagram': 'mdi:instagram',
			'linkedin': 'mdi:linkedin',
			'youtube': 'mdi:youtube',
			'tiktok': 'mdi:tiktok'
		};
		return iconMap[platform?.toLowerCase()] || 'mdi:account';
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

	async function changePassword() {
		if (!isPasswordFormValid.value) {
			passwordMessage.value = 'Please fill in all fields correctly';
			passwordMessageType.value = 'error';
			return;
		}

		passwordChanging.value = true;
		passwordMessage.value = '';
		passwordMessageType.value = '';

		try {
			await waitForClient();
			
			const response = await window.client.changePassword({
				currentPassword: passwordForm.value.currentPassword,
				newPassword: passwordForm.value.newPassword
			});

			if (response.standardResponse.success) {
				passwordMessage.value = 'Password changed successfully!';
				passwordMessageType.value = 'success';
				resetPasswordForm();
			} else {
				passwordMessage.value = response.standardResponse.message || 'Failed to change password';
				passwordMessageType.value = 'error';
			}
		} catch (error) {
			console.error('Error changing password:', error);
			passwordMessage.value = 'Failed to change password: ' + error.message;
			passwordMessageType.value = 'error';
		} finally {
			passwordChanging.value = false;
		}
	}

	function resetPasswordForm() {
		passwordForm.value = {
			currentPassword: '',
			newPassword: '',
			confirmPassword: ''
		};
		passwordMessage.value = '';
		passwordMessageType.value = '';
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
	});
</script>

<style scoped>
	.password-form-grid {
		display: grid;
		grid-template-columns: 200px 1fr;
		gap: 1em;
		align-items: start;
	}

	.password-form-grid label {
		padding-top: 0.75em;
	}

	.password-form-grid input {
		width: 100%;
	}

	.password-form-buttons {
		grid-column: 1 / -1;
		display: flex;
		gap: 1em;
	}

	.password-form-message {
		grid-column: 1 / -1;
	}
</style>
