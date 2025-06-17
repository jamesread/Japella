<template>
	<ul id="nav-section-links">
		<li v-for="link in links" :key="link.name">
			<span v-if="link.type === 'title'" class="nav-title">
				<h2>{{ link.name }}</h2>
			</span>
			<a v-else :href="'#' + link.sectionClass" @click.prevent="showNavSection(link.sectionClass)">
				{{ link.name }}
			</a>
		</li>
	</ul>
</template>

<script setup>
	import { useI18n } from 'vue-i18n'
	import { ref, onMounted } from 'vue';
	const { t } = useI18n()

	const links = ref([
		{
			name: t("nav.post"),
			sectionClass: "post-box",
			type: "link"
		},
		{
			name: t("nav.schedule"),
			type: "title"
		},
		{
			name: t('nav.timeline'),
			sectionClass: 'timeline',
			type: 'link'
		},
		{
			name: t('nav.cannedPosts'),
			sectionClass: 'canned-posts',
			type: 'link',
		},
		{
			name: t('nav.calendar'),
			sectionClass: "calendar",
			type: "link"
		},
		{
			name: t('nav.connections'),
			type: "title",
		},
		{
			name: t('nav.socialaccounts'),
			sectionClass: 'social-accounts',
			type: 'link',
		},
		{
			name: t('nav.system'),
			type: "title",
		},
		{
			name: t('nav.settings'),
			sectionClass: 'settings',
			type: 'link',
		},
		{
			name: t('nav.apikeys'),
			sectionClass: 'settings-apikeys',
			type: 'link',
		},
		{
			name: t('nav.users'),
			sectionClass: 'settings-users',
			type: 'link',
		},
	]);

	function showNavSection(sectionClass) {
		document.getElementById('nav-section-links').querySelectorAll('a').forEach((a) => {
				a.classList.remove('active')
		});

		const link = document.querySelector('a[href="#' + sectionClass + '"]');

		if (link) {
			link.classList.add('active')
		}

		for (const section of document.querySelectorAll('section')) {
			const shown = section.classList.contains(sectionClass)

			section.hidden = !shown
		}
	}
</script>

<style scoped>
li h2 {
	padding-top: 1em;
	padding-left: .2em;
	padding-bottom: .2em;
}
</style>
