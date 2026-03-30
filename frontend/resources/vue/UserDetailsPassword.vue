<template>
	<Section
		:title="pageTitle"
		subtitle="Set a new password for this user account"
		classes="user-details-password"
	>
		<template #toolbar>
			<router-link :to="{ name: 'userDetails', params: { id: String(userId) } }" class="button neutral">
				<Icon icon="material-symbols:arrow-back" />
				Back to user
			</router-link>
			<router-link :to="{ name: 'settingsUsers' }" class="button neutral">
				<Icon icon="material-symbols:group" />
				All users
			</router-link>
		</template>

		<div v-if="!userId" class="inline-notification error">Invalid user ID.</div>
		<div v-else-if="!canResetPassword" class="inline-notification error">
			You do not have permission to reset this user’s password.
		</div>
		<form v-else class="reset-pw-form" @submit.prevent="resetPassword">
			<div class="reset-pw-grid">
				<label for="reset-pw">New password</label>
				<div>
					<input
						id="reset-pw"
						v-model="resetPw"
						type="password"
						autocomplete="new-password"
						required
						minlength="8"
						:disabled="resetPwSaving"
					/>
					<small class="field-hint">At least 8 characters</small>
				</div>
				<label for="reset-pw-confirm">Confirm password</label>
				<input
					id="reset-pw-confirm"
					v-model="resetPwConfirm"
					type="password"
					autocomplete="new-password"
					required
					:disabled="resetPwSaving"
				/>
				<div class="reset-pw-actions">
					<button type="submit" class="good" :disabled="resetPwSaving || !isResetPwValid">
						{{ resetPwSaving ? 'Resetting…' : 'Reset password' }}
					</button>
				</div>
				<div v-if="resetPwMessage" class="span-all">
					<p class="inline-notification" :class="resetPwMessageType">{{ resetPwMessage }}</p>
				</div>
			</div>
		</form>
	</Section>
</template>

<script setup>
	import { ref, computed, watch } from 'vue';
	import { useRoute } from 'vue-router';
	import { Icon } from '@iconify/vue';
	import Section from 'picocrank/vue/components/Section.vue';
	import { waitForClient } from '../javascript/util';

	const route = useRoute();
	const username = ref('');

	const statusPerms = ref([]);
	const statusSuper = ref(false);

	const resetPw = ref('');
	const resetPwConfirm = ref('');
	const resetPwSaving = ref(false);
	const resetPwMessage = ref('');
	const resetPwMessageType = ref('');

	const userId = computed(() => {
		const n = parseInt(String(route.params.id), 10);
		return Number.isFinite(n) && n > 0 ? n : 0;
	});

	const pageTitle = computed(() => (username.value ? `${username.value} — reset password` : 'Reset password'));

	const canResetPassword = computed(
		() => statusSuper.value || (Array.isArray(statusPerms.value) && statusPerms.value.includes('users.reset-password'))
	);

	const isResetPwValid = computed(() => resetPw.value.length >= 8 && resetPw.value === resetPwConfirm.value);

	async function refreshStatus() {
		await waitForClient();
		const st = await window.client.getStatus({});
		statusPerms.value = st.rbacPermissions || [];
		statusSuper.value = Boolean(st.rbacIsSuperuser);
	}

	async function loadUserBasics() {
		if (!userId.value) return;
		try {
			await waitForClient();
			const res = await window.client.getUser({ userId: userId.value });
			username.value = res.user?.username || '';
		} catch {
			username.value = '';
		}
	}

	async function resetPassword() {
		if (!isResetPwValid.value) return;
		resetPwSaving.value = true;
		resetPwMessage.value = '';
		resetPwMessageType.value = '';
		try {
			await waitForClient();
			const res = await window.client.resetUserPassword({
				userId: userId.value,
				newPassword: resetPw.value,
			});
			if (res.standardResponse?.success) {
				resetPw.value = '';
				resetPwConfirm.value = '';
				resetPwMessage.value = res.standardResponse.message || 'Password reset successfully.';
				resetPwMessageType.value = 'success';
			} else {
				resetPwMessage.value = res.standardResponse?.message || 'Failed to reset password.';
				resetPwMessageType.value = 'error';
			}
		} catch (e) {
			resetPwMessage.value = e.message || 'Failed to reset password.';
			resetPwMessageType.value = 'error';
		} finally {
			resetPwSaving.value = false;
		}
	}

	async function load() {
		await refreshStatus();
		await loadUserBasics();
	}

	watch(userId, () => {
		load();
	}, { immediate: true });
</script>

<style scoped>
	.reset-pw-form {
		max-width: 36rem;
		margin-top: 0.5rem;
	}

	.reset-pw-grid {
		display: grid;
		grid-template-columns: minmax(8rem, 180px) 1fr;
		gap: 0.75rem 1rem;
		align-items: start;
	}

	.reset-pw-grid label {
		padding-top: 0.65em;
	}

	.reset-pw-grid input {
		width: 100%;
	}

	.field-hint {
		display: block;
		margin-top: 0.25rem;
		font-size: 0.8em;
		opacity: 0.85;
	}

	.reset-pw-actions {
		grid-column: 1 / -1;
		margin-top: 0.25rem;
	}

	.span-all {
		grid-column: 1 / -1;
	}
</style>
