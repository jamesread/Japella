<template>
	<Section
		title="Create user"
		subtitle="Create a new user account. Password is optional; without one the user cannot log in interactively."
		classes="create-user"
	>
		<template #toolbar>
			<router-link :to="{ name: 'settingsUsers' }" class="button inline-icon neutral">
				<HugeiconsIcon
					:icon="ArrowLeft01Icon"
					width="1em"
					height="1em"
					:strokeWidth="iconStrokeWidth"
					aria-hidden="true"
				/>
				<span>Back to Users</span>
			</router-link>
		</template>

		<FormLayout @submit.prevent="createUser">
			<FormField label="Username" for="new-user-username" :disabled="creating">
				<input
					id="new-user-username"
					v-model="createForm.username"
					type="text"
					autocomplete="off"
					required
					:disabled="creating"
				/>
			</FormField>

			<FormField
				label="Password"
				for="new-user-password"
				:disabled="creating"
				description="Optional. Leave blank to create without interactive login. If set, at least 8 characters."
			>
				<input
					id="new-user-password"
					v-model="createForm.password"
					type="password"
					autocomplete="new-password"
					minlength="8"
					:disabled="creating"
				/>
			</FormField>

			<FormField label="Confirm password" for="new-user-password-confirm" :disabled="creating">
				<input
					id="new-user-password-confirm"
					v-model="createForm.confirmPassword"
					type="password"
					autocomplete="new-password"
					:disabled="creating"
				/>
			</FormField>

			<template #actions>
				<button type="submit" class="inline-icon good" :disabled="creating || !isCreateFormValid">
					<HugeiconsIcon
						v-if="creating"
						:icon="Loading01Icon"
						width="1em"
						height="1em"
						:strokeWidth="iconStrokeWidth"
						aria-hidden="true"
					/>
					<HugeiconsIcon
						v-else
						:icon="UserAdd01Icon"
						width="1em"
						height="1em"
						:strokeWidth="iconStrokeWidth"
						aria-hidden="true"
					/>
					<span>{{ creating ? 'Creating…' : 'Create user' }}</span>
				</button>
				<button type="button" class="neutral" :disabled="creating" @click="resetCreateForm">
					Clear
				</button>
			</template>
		</FormLayout>

		<p v-if="createMessage" class="inline-notification" :class="createMessageType">{{ createMessage }}</p>
	</Section>
</template>

<script setup>
	import { ref, computed } from 'vue';
	import { waitForClient } from '../javascript/util';
	import { HugeiconsIcon } from '@hugeicons/vue';
	import { ArrowLeft01Icon, Loading01Icon, UserAdd01Icon } from '@hugeicons/core-free-icons';
	import Section from 'picocrank/vue/components/Section.vue';
	import FormLayout from 'picocrank/vue/components/FormLayout.vue';
	import FormField from 'picocrank/vue/components/FormField.vue';

	const iconStrokeWidth = 2.5;

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
		if (f.username.trim().length === 0) {
			return false;
		}
		const hasPassword = f.password.length > 0 || f.confirmPassword.length > 0;
		if (!hasPassword) {
			return true;
		}
		return f.password.length >= 8 && f.password === f.confirmPassword;
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
			createMessage.value = 'Enter a username. If setting a password, use matching passwords of at least 8 characters.';
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
