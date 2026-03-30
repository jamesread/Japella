<template>
	<div class="welcome-page">
		<Section
			title="Welcome"
			:subtitle="t('welcome.tagline')"
			classes="welcome-hero"
			id="welcome-hero"
		>
			<p class="welcome-tagline">{{ t('welcome.tagline') }}</p>
			<p class="welcome-lead">{{ t('welcome.gettingstarted') }}</p>
			<router-link :to="{ name: 'postBox' }" class="welcome-primary-cta">
				<HugeiconsIcon :icon="EditIcon" width="22" height="22" aria-hidden="true" />
				<span>{{ t('welcome.cta.post') }}</span>
			</router-link>
		</Section>

		<Section
			:title="t('welcome.section.quickstart')"
			:subtitle="t('welcome.section.quickstart.subtitle')"
			classes="welcome-quickstart"
		>
			<div class="welcome-cards" role="navigation" aria-label="Quick links">
				<router-link
					v-for="card in cards"
					:key="card.routeName"
					:to="{ name: card.routeName }"
					class="welcome-card"
				>
					<div class="welcome-card-icon" aria-hidden="true">
						<HugeiconsIcon :icon="card.icon" :width="cardIconSize" :height="cardIconSize" />
					</div>
					<div class="welcome-card-body">
						<span class="welcome-card-title">{{ card.title }}</span>
						<span class="welcome-card-desc">{{ card.description }}</span>
					</div>
				</router-link>
			</div>
		</Section>
	</div>
</template>

<script setup>
	import { computed } from 'vue'
	import { useI18n } from 'vue-i18n'
	import { HugeiconsIcon } from '@hugeicons/vue'
	import {
		EditIcon,
		ClockIcon,
		UserMultiple03Icon,
		CalendarIcon,
		ImageIcon,
		ActivityIcon,
	} from '@hugeicons/core-free-icons'
	import Section from 'picocrank/vue/components/Section.vue'

	const { t } = useI18n()

	/** Matches picocrank NavigationGrid default icon size */
	const cardIconSize = '2em'

	const cards = computed(() => [
		{
			routeName: 'timeline',
			icon: ClockIcon,
			title: t('welcome.card.timeline.title'),
			description: t('welcome.card.timeline.hint'),
		},
		{
			routeName: 'addSocialAccount',
			icon: UserMultiple03Icon,
			title: t('welcome.card.social.title'),
			description: t('welcome.card.social.hint'),
		},
		{
			routeName: 'calendar',
			icon: CalendarIcon,
			title: t('welcome.card.calendar.title'),
			description: t('welcome.card.calendar.hint'),
		},
		{
			routeName: 'media',
			icon: ImageIcon,
			title: t('welcome.card.media.title'),
			description: t('welcome.card.media.hint'),
		},
		{
			routeName: 'feed',
			icon: ActivityIcon,
			title: t('welcome.card.feed.title'),
			description: t('welcome.card.feed.hint'),
		},
	])
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

	.welcome-primary-cta {
		display: inline-flex;
		align-items: center;
		gap: 0.5rem;
		padding: 0.65rem 1.25rem;
		border-radius: 8px;
		font-weight: 600;
		text-decoration: none;
		color: #fff;
		background: var(--primary-color, #007bff);
		border: 1px solid var(--primary-color, #007bff);
		transition: filter 0.2s ease, transform 0.15s ease, box-shadow 0.2s ease;
		box-shadow: 0 2px 6px rgba(0, 123, 255, 0.25);
	}

	.welcome-primary-cta:hover {
		filter: brightness(1.06);
		transform: translateY(-1px);
		box-shadow: 0 4px 12px rgba(0, 123, 255, 0.3);
	}

	.welcome-primary-cta:focus-visible {
		outline: 2px solid var(--primary-color, #007bff);
		outline-offset: 3px;
	}

	.welcome-cards {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(260px, 1fr));
		gap: 1rem;
		padding-top: 0.25rem;
	}

	.welcome-card {
		display: flex;
		gap: 1rem;
		align-items: flex-start;
		padding: 1.1rem 1rem;
		border: 1px solid var(--border-color, #e1e5e9);
		border-radius: 8px;
		text-decoration: none;
		color: inherit;
		transition: border-color 0.2s ease, transform 0.15s ease, box-shadow 0.2s ease;
		background: transparent;
	}

	.welcome-card:hover {
		border-color: var(--primary-color, #007bff);
		transform: translateY(-2px);
		box-shadow: 0 4px 12px rgba(0, 0, 0, 0.08);
	}

	.welcome-card:focus-visible {
		outline: 2px solid var(--primary-color, #007bff);
		outline-offset: 2px;
	}

	.welcome-card-icon {
		flex-shrink: 0;
		color: var(--primary-color, #007bff);
		display: flex;
		align-items: center;
		justify-content: center;
		margin-top: 0.1rem;
	}

	.welcome-card-body {
		display: flex;
		flex-direction: column;
		gap: 0.35rem;
		min-width: 0;
		text-align: left;
	}

	.welcome-card-title {
		font-weight: 700;
		font-size: 1.05rem;
		line-height: 1.25;
	}

	.welcome-card-desc {
		font-size: 0.875rem;
		line-height: 1.45;
		color: var(--text-muted, #666);
	}

	@media (prefers-color-scheme: dark) {
		.welcome-card:hover {
			box-shadow: 0 4px 14px rgba(0, 0, 0, 0.35);
			background: var(--hover-background-color, rgba(96, 165, 250, 0.06));
		}

		.welcome-card-icon {
			color: var(--primary-color, #60a5fa);
		}

		.welcome-primary-cta {
			box-shadow: 0 2px 10px rgba(96, 165, 250, 0.2);
		}
	}
</style>
