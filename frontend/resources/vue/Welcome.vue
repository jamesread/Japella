<template>
	<section class = "welcome" title = "Welcome to Japella">
		<h2>{{ t('welcome') }}</h2>
		<p>
			{{ t('welcome.gettingstarted') }}
		</p>
	</section>
</template>

<script setup>
	import { useI18n } from 'vue-i18n'
	const { t } = useI18n()
</script>
