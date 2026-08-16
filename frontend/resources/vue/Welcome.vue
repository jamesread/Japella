<template>
	<Navigation ref="welcomeNav">
		<div class="welcome-page">
			<Section
				title="Welcome"
				:subtitle="t('welcome.tagline')"
				classes="welcome-hero"
				id="welcome-hero"
			>
				<p class="welcome-tagline">{{ t('welcome.tagline') }}</p>
				<p class="welcome-lead">{{ t('welcome.gettingstarted') }}</p>
				<button type="button" class="inline-icon neutral" @click="goToPost">
					<HugeiconsIcon
						:icon="EditIcon"
						width="1em"
						height="1em"
						:strokeWidth="iconStrokeWidth"
						aria-hidden="true"
					/>
					<span>{{ t('welcome.cta.post') }}</span>
				</button>
			</Section>

			<Section
				v-for="section in navSections"
				:key="section.name"
				:title="sectionTitle(section.name)"
				:subtitle="sectionSubtitle(section.name)"
				classes="welcome-nav-section"
			>
				<NavigationGrid :filter="sectionFilters[section.name]" />
			</Section>
		</div>
	</Navigation>
</template>

<script setup>
	import { computed, onMounted, ref } from 'vue';
	import { useRouter } from 'vue-router';
	import { useI18n } from 'vue-i18n';
	import { HugeiconsIcon } from '@hugeicons/vue';
	import { EditIcon } from '@hugeicons/core-free-icons';
	import Section from 'picocrank/vue/components/Section.vue';
	import Navigation from 'picocrank/vue/components/Navigation.vue';
	import NavigationGrid from 'picocrank/vue/components/NavigationGrid.vue';
	import { waitForClient } from '../javascript/util';
	import { fetchAppStatus } from '../javascript/status.js';
	import {
		SIDEBAR_NAV_SECTIONS,
		createSectionLinkFilter,
		fetchApprovalsCount,
		setupSidebarNavigation,
		showControlPanelInSidebar,
	} from '../javascript/sidebarNavigation.js';

	const { t } = useI18n();
	const router = useRouter();
	const iconStrokeWidth = 2.5;
	const welcomeNav = ref(null);
	const navSections = SIDEBAR_NAV_SECTIONS;

	const sectionFilters = Object.fromEntries(
		SIDEBAR_NAV_SECTIONS.map((section) => [section.name, createSectionLinkFilter(section)]),
	);

	const navDescriptions = computed(() => ({
		postBox: t('welcome.nav.postBox.hint'),
		approvals: t('welcome.nav.approvals.hint'),
		media: t('welcome.nav.media.hint'),
		campaigns: t('welcome.nav.campaigns.hint'),
		cannedPosts: t('welcome.nav.cannedPosts.hint'),
		calendar: t('welcome.nav.calendar.hint'),
		timeline: t('welcome.nav.timeline.hint'),
		feed: t('welcome.nav.feed.hint'),
		chatBotConversationsAll: t('welcome.nav.conversations.hint'),
		socialAccounts: t('welcome.nav.socialAccounts.hint'),
		chatBots: t('welcome.nav.chatBots.hint'),
		controlPanel: t('welcome.nav.controlPanel.hint'),
	}));

	function sectionTitle(sectionName) {
		const titles = {
			'nav-write': t('welcome.section.write'),
			'nav-read': t('welcome.section.read'),
			'nav-settings': t('welcome.section.settings'),
		};
		return titles[sectionName] || sectionName;
	}

	function sectionSubtitle(sectionName) {
		const subtitles = {
			'nav-write': t('welcome.section.write.subtitle'),
			'nav-read': t('welcome.section.read.subtitle'),
			'nav-settings': t('welcome.section.settings.subtitle'),
		};
		return subtitles[sectionName] || '';
	}

	function goToPost() {
		router.push({ name: 'postBox' });
	}

	async function setupWelcomeNavigation() {
		if (!welcomeNav.value) {
			return;
		}
		await waitForClient();
		const [status, approvalsCount] = await Promise.all([
			fetchAppStatus().catch(() => null),
			fetchApprovalsCount(),
		]);
		setupSidebarNavigation(welcomeNav.value, {
			approvalsCount,
			showControlPanel: showControlPanelInSidebar(status),
			descriptions: navDescriptions.value,
			excludeRoutes: ['postBox'],
		});
	}

	onMounted(() => {
		setupWelcomeNavigation();
	});
</script>

<style scoped>
	.welcome-page {
		max-width: 56rem;
		margin: 0 auto;
	}

	.welcome-tagline {
		margin: 0 0 0.75rem;
		font-size: 1.1rem;
		color: var(--text-muted, #5c6570);
		line-height: 1.45;
		max-width: 36rem;
	}

	.welcome-lead {
		margin: 0 0 1.25rem;
		font-size: 0.95rem;
		line-height: 1.5;
		max-width: 38rem;
	}
</style>
