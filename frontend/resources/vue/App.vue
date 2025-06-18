<template>
	<div id = "layout">
		<aside class = "shown stuck" v-if = "isLoggedIn">
			<SectionNavigation />
		</aside>

		<div id = "content">
			<main v-if = "!isLoggedIn">
				<LoginForm />
			</main>
			<main v-else>
				<Welcome />

				<Timeline />

				<AppStatus />

				<PostBox />

				<Calendar />

				<CannedPosts />

				<SocialAccounts />

				<OAuthServices />
			</main>

			<footer>
				<span><a href = "https://github.com/jamesread/Japella">Japella on GitHub</a></span>
				<span id = "currentVersion"></span>
			</footer>
		</div>
	</div>
</template>

<style scoped>
	footer span {
		display: inline-block;
		margin-right: 10px;
	}
</style>

<script setup>
	import { waitForClient } from '../javascript/util.js'
	import { ref, onMounted } from 'vue';

	const clientReady = ref(false);
	const isLoggedIn = ref(false);

	async function getStatus() {
		const st = await window.client.getStatus();

		if (st.isLoggedIn) {
			isLoggedIn.value = true;

			window.dispatchEvent(new CustomEvent('status-updated', {
				"detail": st
			}));
		} else {
			isLoggedIn.value = false;
		}

		document.getElementById('currentVersion').innerText = 'Version: ' + status.version;
	}

	onMounted(async () => {
		await waitForClient();
		clientReady.value = true;
	});
</script>
