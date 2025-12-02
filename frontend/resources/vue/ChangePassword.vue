<template>
	<Section
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
					<small style="display: block; margin-top: 0.25em; font-size: 0.8em;">
						Password must be at least 8 characters long
					</small>
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
</template>

<script setup>
	import { ref, computed } from 'vue';
	import { Icon } from '@iconify/vue';
	import Section from 'picocrank/vue/components/Section.vue';
	import { waitForClient } from '../javascript/util';

	const passwordForm = ref({
		currentPassword: '',
		newPassword: '',
		confirmPassword: ''
	});
	const passwordChanging = ref(false);
	const passwordMessage = ref('');
	const passwordMessageType = ref('');

	const isPasswordFormValid = computed(() => {
		return passwordForm.value.currentPassword.length > 0 &&
			   passwordForm.value.newPassword.length >= 8 &&
			   passwordForm.value.confirmPassword.length > 0 &&
			   passwordForm.value.newPassword === passwordForm.value.confirmPassword;
	});

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


