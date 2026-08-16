<template>
	<Section
		:title="pageTitle"
		subtitle="Set a new password for this user account"
		classes="user-details-password"
	>
		<template #toolbar>
			<router-link :to="{ name: 'userDetails', params: { id: String(userId) } }" class="button inline-icon neutral">
				<HugeiconsIcon
					:icon="ArrowLeft01Icon"
					width="1em"
					height="1em"
					:strokeWidth="iconStrokeWidth"
					aria-hidden="true"
				/>
				<span>Back to user</span>
			</router-link>
			<router-link :to="{ name: 'settingsUsers' }" class="button inline-icon neutral">
				<HugeiconsIcon
					:icon="UserGroupIcon"
					width="1em"
					height="1em"
					:strokeWidth="iconStrokeWidth"
					aria-hidden="true"
				/>
				<span>All users</span>
			</router-link>
		</template>

		<div v-if="!userId" class="inline-notification error">Invalid user ID.</div>
		<div v-else-if="!canResetPassword" class="inline-notification error">
			You do not have permission to reset this user’s password.
		</div>
		<FormLayout v-else @submit.prevent="resetPassword">
			<FormField
				label="New password"
				for="reset-pw"
				:disabled="resetPwSaving"
				description="At least 8 characters."
			>
				<input
					id="reset-pw"
					v-model="resetPw"
					type="password"
					autocomplete="new-password"
					required
					minlength="8"
					:disabled="resetPwSaving"
				/>
			</FormField>

			<FormField label="Confirm password" for="reset-pw-confirm" :disabled="resetPwSaving">
				<input
					id="reset-pw-confirm"
					v-model="resetPwConfirm"
					type="password"
					autocomplete="new-password"
					required
					:disabled="resetPwSaving"
				/>
			</FormField>

			<template #actions>
				<button type="submit" class="good" :disabled="resetPwSaving || !isResetPwValid">
					{{ resetPwSaving ? 'Resetting…' : 'Reset password' }}
				</button>
			</template>
		</FormLayout>

		<p v-if="resetPwMessage" class="inline-notification" :class="resetPwMessageType">{{ resetPwMessage }}</p>
	</Section>
</template>

<script setup>
	import { ref, computed, watch } from 'vue';
	import { useRoute } from 'vue-router';
	import { HugeiconsIcon } from '@hugeicons/vue';
	import { ArrowLeft01Icon, UserGroupIcon } from '@hugeicons/core-free-icons';
	import Section from 'picocrank/vue/components/Section.vue';
	import FormLayout from 'picocrank/vue/components/FormLayout.vue';
	import FormField from 'picocrank/vue/components/FormField.vue';
	import { waitForClient } from '../javascript/util';

	const iconStrokeWidth = 2.5;
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
