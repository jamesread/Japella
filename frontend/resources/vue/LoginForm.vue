<template>
	<section class = "small login-form">
		<h2>{{ t('section.login.title') }}</h2>
		<p>{{ t('section.login.description') }}</p>
		<Login 
			ref="loginRef"
			:show-default-tabs="false"
			:custom-tabs="customTabs"
			@local-login="handleLocalLogin"
		>
			<template #tab-local>
				<div class="login-section">
					<form @submit.prevent="handleLocalLogin" class="local-login-form">
						<input
							id="username"
							v-model="localLogin.username"
							required
							autocomplete="username"
							placeholder="Username"
						/>
						<input
							id="password"
							type="password"
							v-model="localLogin.password"
							required
							autocomplete="current-password"
							placeholder="Password"
						/>
						<button type="submit" class="good">{{ t('section.login.login') }}</button>
						<div v-if="localLoginError" class="error-message">
							{{ localLoginError }}
						</div>
					</form>
				</div>
			</template>
		</Login>
	</section>
</template>


<style scoped>
	section {
		max-width: 320px;
	}

	.login-section {
		display: flex;
		flex-direction: column;
		gap: 1rem;
		align-items: center;
	}

	.local-login-form {
		display: flex;
		flex-direction: column;
		gap: 1rem;
		width: 100%;
	}

	.local-login-form input {
		padding: 0.75rem;
		border: 1px solid var(--border-color, #ccc);
		border-radius: 4px;
		font-size: 1em;
		background-color: var(--input-bg, #fff);
		color: var(--text-color, #000);
		width: 100%;
		box-sizing: border-box;
	}

	.local-login-form input:focus {
		outline: none;
		border-color: var(--primary-color, #007bff);
		box-shadow: 0 0 0 2px rgba(0, 123, 255, 0.25);
	}

	.local-login-form button {
		width: 100%;
		padding: 0.75rem 1.5rem;
		font-size: 1em;
		cursor: pointer;
		border: none;
		border-radius: 4px;
		transition: opacity 0.2s;
	}

	.local-login-form button:hover {
		opacity: 0.9;
	}

	.error-message {
		padding: 0.75rem;
		background-color: var(--error-bg, #fee);
		color: var(--error-color, #c33);
		border-radius: 4px;
		font-size: 0.9em;
		margin-top: 0.5rem;
		width: 100%;
	}
</style>

<script setup>
	import { ref, reactive } from 'vue'
	import { useI18n } from 'vue-i18n'
	import Login from 'picocrank/vue/components/Login.vue'

	const emit = defineEmits(['login-success'])

	const { t } = useI18n()

	// Only show local login tab (hide OAuth tab)
	const customTabs = ref([
		{ id: 'local', label: 'Username & Password' }
	])

	// Local login state (for the form inputs)
	const localLogin = reactive({
		username: '',
		password: ''
	})

	const localLoginError = ref('')

	async function handleLocalLogin(credentials) {
		// Clear any previous errors
		localLoginError.value = ''

		// Use credentials from event if provided, otherwise use local state
		const username = credentials?.username || localLogin.username
		const password = credentials?.password || localLogin.password

		try {
			let res = await window.client.loginWithUsernameAndPassword({
				"username": username,
				"password": password
			})

			if (res.standardResponse.success) {
				console.log('Login successful:', res)
				// Reset form
				localLogin.username = ''
				localLogin.password = ''
				emit('login-success', res)
			} else {
				const errorMsg = t('section.login.error') + ': ' + res.standardResponse.errorMessage
				localLoginError.value = errorMsg
			}
		} catch (error) {
			const errorMsg = t('section.login.error')
			localLoginError.value = errorMsg
		}
	}
</script>
