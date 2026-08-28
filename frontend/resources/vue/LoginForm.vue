<template>
	<section class="small login-form">
		<h2>{{ t('section.login.title') }}</h2>
		<p>{{ t('section.login.description') }}</p>
		<Login
			ref="loginRef"
			local-only
			@local-login="handleLocalLogin"
		/>
	</section>
</template>


<style scoped>
	section {
		max-width: 384px;
	}

	.login-form :deep(.login-container) {
		max-width: 100%;
		margin-top: 0;
		margin-bottom: 0;
	}
</style>

<script setup>
	import { ref } from 'vue'
	import { useI18n } from 'vue-i18n'
	import Login from 'picocrank/vue/components/Login.vue'

	const emit = defineEmits(['login-success'])

	const { t } = useI18n()

	const loginRef = ref(null)

	async function handleLocalLogin(credentials) {
		try {
			const res = await window.client.loginWithUsernameAndPassword({
				username: credentials.username,
				password: credentials.password,
			})

			if (res.standardResponse?.success) {
				loginRef.value?.resetLocalForm()
				emit('login-success', res)
			} else {
				loginRef.value?.setLocalLoginError(
					res.standardResponse?.message || t('section.login.error')
				)
			}
		} catch {
			loginRef.value?.setLocalLoginError(t('section.login.error'))
		}
	}
</script>
