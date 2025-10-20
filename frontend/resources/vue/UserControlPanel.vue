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

		<div v-if="errorMessage" class="error-section">
			<p class="inline-notification error">{{ errorMessage }}</p>
		</div>

		<div v-if="loading" class="loading-section">
			<div class="icon-and-text">
				<Icon icon="eos-icons:loading" width="24" height="24" />
				<span>Loading user data...</span>
			</div>
		</div>

		<div v-else-if="userData" class="user-profile">
			<!-- User Profile Card -->
			<div class="profile-card">
				<div class="profile-header">
					<div class="profile-avatar">
						<Icon icon="mdi:account-circle" width="64" height="64" />
					</div>
					<div class="profile-info">
						<h3>{{ userData.username }}</h3>
						<p class="profile-email">{{ userData.email || 'No email provided' }}</p>
						<p class="profile-role">{{ userData.role || 'User' }}</p>
					</div>
				</div>
			</div>

			<!-- Account Information -->
			<div class="account-section">
				<h4>Account Information</h4>
				<div class="info-grid">
					<div class="info-item">
						<label>Username:</label>
						<span>{{ userData.username }}</span>
					</div>
					<div class="info-item">
						<label>Email:</label>
						<span>{{ userData.email || 'Not provided' }}</span>
					</div>
					<div class="info-item">
						<label>Role:</label>
						<span>{{ userData.role || 'User' }}</span>
					</div>
					<div class="info-item">
						<label>Account Created:</label>
						<span>{{ formatDate(userData.createdAt) }}</span>
					</div>
					<div class="info-item">
						<label>Last Login:</label>
						<span>{{ formatDate(userData.lastLoginAt) }}</span>
					</div>
				</div>
			</div>

			<!-- Social Accounts -->
			<div class="social-accounts-section">
				<h4>Connected Social Accounts</h4>
				<div v-if="socialAccounts.length === 0" class="no-accounts">
					<p class="inline-notification note">No social accounts connected.</p>
				</div>
				<div v-else class="social-accounts-grid">
					<div v-for="account in socialAccounts" :key="account.id" class="social-account-card">
						<div class="account-header">
							<Icon :icon="getSocialIcon(account.platform)" width="24" height="24" />
							<span class="account-platform">{{ account.platform }}</span>
							<span class="account-status" :class="account.isActive ? 'active' : 'inactive'">
								{{ account.isActive ? 'Active' : 'Inactive' }}
							</span>
						</div>
						<div class="account-details">
							<p><strong>Handle:</strong> {{ account.handle || account.identity || 'Unknown' }}</p>
							<p><strong>Connected:</strong> {{ formatDate(account.createdAt) }}</p>
						</div>
					</div>
				</div>
			</div>

			<!-- Password Change Section -->
			<div class="password-change-section">
				<h4>Change Password</h4>
				<form @submit.prevent="changePassword" class="password-form">
					<div class="form-group">
						<label for="current-password">Current Password:</label>
						<input 
							type="password" 
							id="current-password" 
							v-model="passwordForm.currentPassword" 
							required 
							:disabled="passwordChanging"
						/>
					</div>
					<div class="form-group">
						<label for="new-password">New Password:</label>
						<input 
							type="password" 
							id="new-password" 
							v-model="passwordForm.newPassword" 
							required 
							minlength="8"
							:disabled="passwordChanging"
						/>
						<small class="password-hint">Password must be at least 8 characters long</small>
					</div>
					<div class="form-group">
						<label for="confirm-password">Confirm New Password:</label>
						<input 
							type="password" 
							id="confirm-password" 
							v-model="passwordForm.confirmPassword" 
							required 
							:disabled="passwordChanging"
						/>
					</div>
					<div class="form-actions">
						<button type="submit" :disabled="passwordChanging || !isPasswordFormValid" class="change-password-btn">
							<Icon v-if="passwordChanging" icon="eos-icons:loading" width="16" height="16" />
							<Icon v-else icon="mdi:lock-reset" width="16" height="16" />
							<span>{{ passwordChanging ? 'Changing...' : 'Change Password' }}</span>
						</button>
						<button type="button" @click="resetPasswordForm" :disabled="passwordChanging" class="reset-btn">
							Reset
						</button>
					</div>
					<div v-if="passwordMessage" class="password-message" :class="passwordMessageType">
						{{ passwordMessage }}
					</div>
				</form>
			</div>

			<!-- Quick Actions -->
			<div class="quick-actions-section">
				<h4>Quick Actions</h4>
				<div class="actions-grid">
					<button @click="goToSocialAccounts" class="action-button">
						<Icon icon="mdi:account-multiple" />
						<span>Manage Social Accounts</span>
					</button>
					<button @click="goToSettings" class="action-button">
						<Icon icon="mdi:cog" />
						<span>Account Settings</span>
					</button>
					<button @click="goToApiKeys" class="action-button">
						<Icon icon="mdi:key" />
						<span>API Keys</span>
					</button>
				</div>
			</div>

			<!-- Logout Section -->
			<div class="logout-section">
				<h4>Session Management</h4>
				<div class="logout-card">
					<div class="logout-info">
						<Icon icon="mdi:logout" width="24" height="24" />
						<div class="logout-text">
							<h5>Sign Out</h5>
							<p>End your current session and return to the login page.</p>
						</div>
					</div>
					<button @click="logout" :disabled="loggingOut" class="logout-button">
						<Icon v-if="loggingOut" icon="eos-icons:loading" width="16" height="16" />
						<Icon v-else icon="mdi:logout" width="16" height="16" />
						<span>{{ loggingOut ? 'Signing Out...' : 'Sign Out' }}</span>
					</button>
				</div>
			</div>
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
	.user-control-panel {
		max-width: 1000px;
		margin: 0 auto;
	}

	.loading-section {
		text-align: center;
		padding: 2em;
	}

	.profile-card {
		background-color: #2a2a2a;
		border-radius: 0.5em;
		padding: 1.5em;
		margin-bottom: 2em;
	}

	.profile-header {
		display: flex;
		align-items: center;
		gap: 1em;
	}

	.profile-avatar {
		color: #666;
	}

	.profile-info h3 {
		margin: 0 0 0.5em 0;
		color: white;
		font-size: 1.5em;
	}

	.profile-email {
		margin: 0 0 0.25em 0;
		color: #ccc;
	}

	.profile-role {
		margin: 0;
		color: #999;
		font-size: 0.9em;
	}

	.account-section,
	.social-accounts-section,
	.password-change-section,
	.quick-actions-section,
	.logout-section {
		margin-bottom: 2em;
	}

	.account-section h4,
	.social-accounts-section h4,
	.password-change-section h4,
	.quick-actions-section h4,
	.logout-section h4 {
		color: white;
		margin-bottom: 1em;
		border-bottom: 1px solid #444;
		padding-bottom: 0.5em;
	}

	.password-form {
		background-color: #2a2a2a;
		border-radius: 0.5em;
		padding: 1.5em;
	}

	.form-group {
		margin-bottom: 1em;
	}

	.form-group label {
		display: block;
		margin-bottom: 0.5em;
		color: #ccc;
		font-weight: bold;
	}

	.form-group input {
		width: 100%;
		padding: 0.75em;
		border: 1px solid #666;
		border-radius: 0.25em;
		background-color: #1a1a1a;
		color: white;
		font-size: 1em;
	}

	.form-group input:focus {
		outline: none;
		border-color: #4CAF50;
		box-shadow: 0 0 0 2px rgba(76, 175, 80, 0.2);
	}

	.form-group input:disabled {
		background-color: #333;
		color: #666;
		cursor: not-allowed;
	}

	.password-hint {
		display: block;
		margin-top: 0.25em;
		color: #999;
		font-size: 0.8em;
	}

	.form-actions {
		display: flex;
		gap: 1em;
		margin-top: 1.5em;
	}

	.change-password-btn {
		display: flex;
		align-items: center;
		gap: 0.5em;
		padding: 0.75em 1.5em;
		background-color: #4CAF50;
		border: none;
		border-radius: 0.25em;
		color: white;
		font-weight: bold;
		cursor: pointer;
		transition: background-color 0.2s ease;
	}

	.change-password-btn:hover:not(:disabled) {
		background-color: #45a049;
	}

	.change-password-btn:disabled {
		background-color: #666;
		cursor: not-allowed;
	}

	.reset-btn {
		padding: 0.75em 1.5em;
		background-color: transparent;
		border: 1px solid #666;
		border-radius: 0.25em;
		color: white;
		cursor: pointer;
		transition: all 0.2s ease;
	}

	.reset-btn:hover:not(:disabled) {
		background-color: #666;
	}

	.reset-btn:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}

	.password-message {
		margin-top: 1em;
		padding: 0.75em;
		border-radius: 0.25em;
		font-weight: bold;
	}

	.password-message.success {
		background-color: #4CAF50;
		color: white;
	}

	.password-message.error {
		background-color: #f44336;
		color: white;
	}

	.logout-card {
		background-color: #2a2a2a;
		border-radius: 0.5em;
		padding: 1.5em;
		display: flex;
		align-items: center;
		justify-content: space-between;
		border-left: 4px solid #f44336;
	}

	.logout-info {
		display: flex;
		align-items: center;
		gap: 1em;
		flex-grow: 1;
	}

	.logout-info svg {
		color: #f44336;
		flex-shrink: 0;
	}

	.logout-text h5 {
		margin: 0 0 0.25em 0;
		color: white;
		font-size: 1.1em;
	}

	.logout-text p {
		margin: 0;
		color: #ccc;
		font-size: 0.9em;
	}

	.logout-button {
		display: flex;
		align-items: center;
		gap: 0.5em;
		padding: 0.75em 1.5em;
		background-color: #f44336;
		border: none;
		border-radius: 0.25em;
		color: white;
		font-weight: bold;
		cursor: pointer;
		transition: background-color 0.2s ease;
		flex-shrink: 0;
	}

	.logout-button:hover:not(:disabled) {
		background-color: #d32f2f;
	}

	.logout-button:disabled {
		background-color: #666;
		cursor: not-allowed;
	}

	.info-grid {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
		gap: 1em;
	}

	.info-item {
		display: flex;
		justify-content: space-between;
		padding: 0.75em;
		background-color: #2a2a2a;
		border-radius: 0.25em;
	}

	.info-item label {
		font-weight: bold;
		color: #ccc;
	}

	.info-item span {
		color: white;
	}

	.social-accounts-grid {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
		gap: 1em;
	}

	.social-account-card {
		background-color: #2a2a2a;
		border-radius: 0.5em;
		padding: 1em;
		border-left: 4px solid #666;
	}

	.account-header {
		display: flex;
		align-items: center;
		gap: 0.5em;
		margin-bottom: 0.75em;
	}

	.account-platform {
		font-weight: bold;
		color: white;
		text-transform: capitalize;
	}

	.account-status {
		margin-left: auto;
		padding: 0.25em 0.5em;
		border-radius: 0.25em;
		font-size: 0.8em;
		font-weight: bold;
	}

	.account-status.active {
		background-color: #4caf50;
		color: white;
	}

	.account-status.inactive {
		background-color: #f44336;
		color: white;
	}

	.account-details p {
		margin: 0.25em 0;
		color: #ccc;
		font-size: 0.9em;
	}

	.actions-grid {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
		gap: 1em;
	}

	.action-button {
		display: flex;
		align-items: center;
		gap: 0.5em;
		padding: 1em;
		background-color: #2a2a2a;
		border: 1px solid #666;
		border-radius: 0.5em;
		color: white;
		cursor: pointer;
		transition: all 0.2s ease;
	}

	.action-button:hover {
		background-color: #3a3a3a;
		border-color: #888;
	}

	.action-button svg {
		flex-shrink: 0;
	}

	.no-accounts {
		text-align: center;
		padding: 2em;
	}
</style>
