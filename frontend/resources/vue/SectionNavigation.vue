<template>
	<ul id="nav-section-links">
		<li v-for="link in links" :key="link.name">
			<span v-if="link.type === 'title'" class="nav-title">
				<h2>{{ link.name }}</h2>
			</span>
			<a v-else :href="'#' + link.sectionName" @click.prevent="showNavSection(link.sectionName)">
				{{ link.name }}
			</a>
		</li>
	</ul>
</template>

<script setup>
	import { useI18n } from 'vue-i18n'
	import { ref, onMounted } from 'vue';
	const { t } = useI18n()

	const emit = defineEmits(['section-change-request'])

	const links = ref([
		{
			name: t("nav.post"),
			sectionName: "postBox",
			type: "link"
		},
		{
			name: t("nav.schedule"),
			type: "title"
		},
		{
			name: t('nav.timeline'),
			sectionName: 'timeline',
			type: 'link'
		},
		{
			name: t('nav.cannedPosts'),
			sectionName: 'cannedPosts',
			type: 'link',
		},
		{
			name: t('nav.calendar'),
			sectionName: "calendar",
			type: "link"
		},
		{
			name: t('nav.connections'),
			type: "title",
		},
		{
			name: t('nav.socialaccounts'),
			sectionName: 'socialAccounts',
			type: 'link',
		},
		{
			name: t('nav.system'),
			type: "title",
		},
		{
			name: t('nav.settings'),
			sectionName: 'settings',
			type: 'link',
		},
		{
			name: t('nav.apikeys'),
			sectionName: 'settingsApiKeys',
			type: 'link',
		},
		{
			name: t('nav.users'),
			sectionName: 'settingsUsers',
			type: 'link',
		},
	]);

	function showNavSection(sectionName) {
		emit('section-change-request', sectionName);
	}

	defineExpose({
		showNavSection
	})
</script>

<style scoped>
li h2 {
	padding-top: 1em;
	padding-left: .2em;
	padding-bottom: .2em;
}
</style>
