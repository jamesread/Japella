<template>
	<section class = "small login-form">
		<h2>{{ t('section.login.title') }}</h2>
		<p>{{ t('section.login.description') }}</p>
		<form @submit.prevent="login">
			<input type="text" required placeholder = "Username" v-model = "username" />

			<input type="password" required placeholder = "password" v-model = "password" />

			<button type="submit">Login</button>
		</form>

		<br />

		<div class = "flex-row button-group">
			<button class = "neutral" @click = "forgotPassword">{{ t('section.login.forgot-password') }}</button>
			<button class = "neutral">{{ t('section.login.create-account') }}</button>
		</div>
	</section>
</template>


<style scoped>
	section {
		max-width: 320px;
	}

	form {
		grid-template-columns: 1fr;
	}

	.button-group {
		gap: 1em;
	}
</style>

<script setup>
	import { onMounted, ref } from 'vue'
	import { useI18n } from 'vue-i18n'

	const emit = defineEmits(['login-success'])

	const { t } = useI18n()

	const username = ref(null)
	const password = ref(null)

	async function login() {
		console.log('Login attempt:', username.value, password.value)

		let res = await window.client.loginWithUsernameAndPassword({
			"username": username.value,
			"password": password.value
		})

		if (res.standardResponse.success) {
			console.log('Login successful:', res)
			emit('login-success', res)
		} else {
			console.error('Login failed:', res)
			alert(t('section.login.error'))
		}
	}

	function forgotPassword() {
		alert(t('section.login.forgot-password-not-implemented'))
	}
</script>
