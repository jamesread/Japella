<template>
	<ul id="nav-section-links">
		<li v-for="link in links" :key="link.name">
			<span v-if="link.type === 'title'" class="nav-title">
				<h2>{{ link.name }}</h2>
			</span>
			<a v-else :href="'#' + link.sectionName" @click.prevent="showNavSection(link.sectionName)">
				<Icon :icon="link.icon" v-if="link.icon" />
				{{ link.name }}
			</a>
		</li>
	</ul>
</template>

<script setup>
	import { useI18n } from 'vue-i18n'
	import { ref } from 'vue';
	const { t } = useI18n()
	import { Icon } from '@iconify/vue';
	import { inject } from 'vue';

	const changeSection = inject('changeSection');

	const links = ref([
		{
			name: t("nav.post"),
			sectionName: "postBox",
			icon: "jam:write-f",
			type: "link"
		},
		{
			name: t("nav.media"),
			sectionName: "media",
			icon: "jam:picture-f",
			type: "link"
		},
		{
			name: t("nav.schedule"),
			type: "title"
		},
		{
			name: t('nav.campaigns'),
			sectionName: 'campaigns',
			type: 'link',
			icon: "jam:volume-up",
		},
		{
			name: t('nav.cannedPosts'),
			sectionName: 'cannedPosts',
			type: 'link',
			icon: "jam:box",
		},
		{
			name: t('nav.timeline'),
			sectionName: 'timeline',
			type: 'link',
			icon: "jam:clock",
		},
		{
			name: t('nav.calendar'),
			sectionName: "calendar",
			type: "link",
			icon: "jam:calendar-alt",
		},
		{
			name: t('nav.connections'),
			type: "title",
		},
		{
			name: t('nav.socialaccounts'),
			sectionName: 'socialAccounts',
			type: 'link',
			icon: "jam:users",
		},
		{
			name: t('nav.chataccounts'),
			sectionName: 'oauthServices',
			type: 'link',
			icon: "jam:message",
		},
		{
			name: t('nav.system'),
			type: "title",
		},
		{
			name: t('nav.settings'),
			sectionName: 'settings',
			type: 'link',
			icon: "jam:settings-alt",
		},
		{
			name: t('nav.apikeys'),
			sectionName: 'settingsApiKeys',
			type: 'link',
			icon: "jam:key",
		},
		{
			name: t('nav.users'),
			sectionName: 'settingsUsers',
			type: 'link',
			icon: "jam:users",
		},
	]);

	function showNavSection(sectionName) {
		changeSection(sectionName);
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
