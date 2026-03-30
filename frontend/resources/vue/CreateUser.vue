<template>
	<Section
		title="Create user"
		subtitle="Create a new user account."
		classes="create-user"
	>
		<template #toolbar>
			<router-link :to="{ name: 'settingsUsers' }" class="button neutral">
				<Icon icon="material-symbols:arrow-back" />
				Back to Users
			</router-link>
		</template>

		<form class="create-user-panel" @submit.prevent="createUser">
			<div class="create-user-grid">
				<label for="new-user-username">Username</label>
				<input
					id="new-user-username"
					v-model="createForm.username"
					type="text"
					autocomplete="off"
					required
					:disabled="creating"
				/>

				<label for="new-user-password">Password</label>
				<div>
					<input
						id="new-user-password"
						v-model="createForm.password"
						type="password"
						autocomplete="new-password"
						required
						minlength="8"
						:disabled="creating"
					/>
					<small class="field-hint">At least 8 characters (same rule as password change)</small>
				</div>

				<label for="new-user-password-confirm">Confirm password</label>
				<input
					id="new-user-password-confirm"
					v-model="createForm.confirmPassword"
					type="password"
					autocomplete="new-password"
					required
					:disabled="creating"
				/>

				<div class="create-user-actions">
					<button type="submit" class="good" :disabled="creating || !isCreateFormValid">
						<Icon v-if="creating" icon="eos-icons:loading" width="16" height="16" />
						<Icon v-else icon="material-symbols:person-add" width="16" height="16" />
						<span>{{ creating ? 'Creating…' : 'Create user' }}</span>
					</button>
					<button type="button" class="neutral" :disabled="creating" @click="resetCreateForm">
						Clear
					</button>
				</div>

				<div v-if="createMessage" class="create-user-message span-all">
					<p class="inline-notification" :class="createMessageType">{{ createMessage }}</p>
				</div>
			</div>
		</form>
	</Section>
</template>

<script setup>
	import { ref, computed } from 'vue';
	import { useRouter } from 'vue-router';
	import { waitForClient } from '../javascript/util';
	import { Icon } from '@iconify/vue';
	import Section from 'picocrank/vue/components/Section.vue';

	const router = useRouter();
	const creating = ref(false);
	const createForm = ref({
		username: '',
		password: '',
		confirmPassword: ''
	});
	const createMessage = ref('');
	const createMessageType = ref('');

	const isCreateFormValid = computed(() => {
		const f = createForm.value;
		return (
			f.username.trim().length > 0 &&
			f.password.length >= 8 &&
			f.password === f.confirmPassword
		);
	});

	function clearCreateFormFields() {
		createForm.value = {
			username: '',
			password: '',
			confirmPassword: ''
		};
	}

	function resetCreateForm() {
		clearCreateFormFields();
		createMessage.value = '';
		createMessageType.value = '';
	}

	async function createUser() {
		if (!isCreateFormValid.value) {
			createMessage.value = 'Enter a username, matching passwords of at least 8 characters.';
			createMessageType.value = 'error';
			return;
		}

		creating.value = true;
		createMessage.value = '';
		createMessageType.value = '';

		try {
			await waitForClient();
			const res = await window.client.createUser({
				username: createForm.value.username.trim(),
				password: createForm.value.password
			});

			if (res.standardResponse?.success) {
				createMessage.value = res.standardResponse.message || 'User created successfully.';
				createMessageType.value = 'success';
				clearCreateFormFields();
			} else {
				createMessage.value = res.standardResponse?.message || 'Could not create user.';
				createMessageType.value = 'error';
			}
		} catch (error) {
			console.error('Error creating user:', error);
			createMessage.value = error.message || 'Failed to create user.';
			createMessageType.value = 'error';
		} finally {
			creating.value = false;
		}
	}
</script>

<style scoped>
	.create-user-panel {
		max-width: 36rem;
	}

	.create-user-grid {
		display: grid;
		grid-template-columns: minmax(8rem, 180px) 1fr;
		gap: 0.75rem 1rem;
		align-items: start;
	}

	.create-user-grid label {
		padding-top: 0.65em;
	}

	.create-user-grid input {
		width: 100%;
	}

	.field-hint {
		display: block;
		margin-top: 0.25rem;
		font-size: 0.8em;
		opacity: 0.85;
	}

	.create-user-actions {
		grid-column: 1 / -1;
		display: flex;
		flex-wrap: wrap;
		gap: 0.75rem;
		align-items: center;
		margin-top: 0.25rem;
	}

	.create-user-actions button {
		display: inline-flex;
		align-items: center;
		gap: 0.35rem;
	}

	.create-user-message.span-all {
		grid-column: 1 / -1;
		margin: 0;
	}
</style>
