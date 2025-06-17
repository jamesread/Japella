<template>
	<div id = "layout">
		<aside class = "shown stuck">
			<SectionNavigation />
		</aside>

		<div id = "content">
			<main>
				<Welcome />

				<Timeline />

				<AppStatus />

				<PostBox />

				<Calendar />

				<CannedPosts />

				<SocialAccounts />

				<OAuthServices />

				<LoginForm />
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
