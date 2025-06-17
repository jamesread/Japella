<template>
	<section class = "small login-form">
		<h1>Login</h1>
		<form @submit.prevent="login">
			<label for="username">Username:</label>
			<input type="text" id="username" required placeholder = "Alice" />

			<label for="password">Password:</label>
			<input type="password" id="password" required />

			<button type="submit">Login</button>
		</form>
	</section>
</template>


<style scoped>
	form {
		grid-template-columns: max-content 1fr;
	}
</style>
