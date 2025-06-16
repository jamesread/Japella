<template>
	<section class = "welcome" title = "Welcome to Japella">
		<h2>Welcome to Japella</h2>
		<p>
			To get started, please select an option from the menu above.
		</p>
	</section>

	<AppStatus />

	<PostBox />

	<Calendar />

	<CannedPosts />

	<SocialAccounts />

	<OAuthServices />
</template>

<script setup>
	import AppStatus from './app-status.vue'
	import PostBox from './post-box.vue'
	import Calendar from './calendar.vue'
	import CannedPosts from './canned-posts.vue'
	import SocialAccounts from './social-accounts.vue'
	import OAuthServices from './oauth-services.vue'
</script>
