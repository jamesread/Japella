<template>
	<section class = "small login-form">
		<h1>{{ t('section.login.title') }}</h1>
		<p>{{ t('section.login.description') }}</p>
		<form @submit.prevent="login">
			<label for="username">Username:</label>
			<input type="text" required placeholder = "Alice" v-model = "username" />

			<label for="password">Password:</label>
			<input type="password" required v-model = "password" />

			<button type="submit">Login</button>
		</form>
	</section>
</template>


<style scoped>
	form {
		grid-template-columns: max-content 1fr;
	}
</style>

<script setup>
	import { onMounted, ref } from 'vue'
	import { useI18n } from 'vue-i18n'
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
		} else {
			console.error('Login failed:', res)
			alert(t('section.login.error'))
		}

	}
</script>
