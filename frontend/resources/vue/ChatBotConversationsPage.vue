<template>
	<Section
		title="Conversations"
		:subtitle="bot ? `${bot.name} · ${bot.connector} · ${displayIdentity}` : 'Chat bot messages'"
		classes="chat-bot-conversations-page"
	>
		<template #toolbar>
			<router-link :to="{ name: 'chatBotDetails', params: detailsParams }" class="button neutral">
				<Icon icon="material-symbols:arrow-back" />
				Back to bot details
			</router-link>
		</template>

		<div v-if="!clientReady || loading">
			<p>Loading...</p>
		</div>
		<div v-else-if="error">
			<p class="inline-notification error">{{ error }}</p>
		</div>
		<div v-else-if="!bot">
			<p class="inline-notification note">Bot not found.</p>
		</div>
		<div v-else>
			<p class="bot-meta-row">
				<Icon :icon="protocolIcon" width="18" height="18" />
				<span class="bot-meta-connector">{{ bot.connector }}</span>
				<span class="bot-meta-sep">·</span>
				<span class="bot-meta-identity">Identity: {{ displayIdentity }}</span>
			</p>
			<ChatBotConversations :bot="bot" :route-identity="routeIdentityRaw" />
		</div>
	</Section>
</template>

<script setup>
	import { computed, ref, onMounted } from 'vue';
	import { useRoute } from 'vue-router';
	import { Icon } from '@iconify/vue';
	import Section from 'picocrank/vue/components/Section.vue';
	import { waitForClient } from '../javascript/util';
	import ChatBotConversations from './ChatBotConversations.vue';

	const route = useRoute();
	const clientReady = ref(false);
	const loading = ref(true);
	const error = ref('');
	const connector = ref('');
	const routeIdentityRaw = ref('');
	const bot = ref(null);

	const detailsParams = computed(() => ({
		connector: route.params.connector
			? decodeURIComponent(String(route.params.connector))
			: connector.value,
		identity:
			route.params.identity !== undefined &&
			route.params.identity !== null &&
			String(route.params.identity) !== ''
				? decodeURIComponent(String(route.params.identity))
				: routeIdentityRaw.value || '_',
	}));
	const displayIdentity = computed(() => String(bot.value?.identity || 'N/A'));
	const protocolIcon = computed(() => {
		const connectorName = String(bot.value?.connector || '').toLowerCase();
		if (connectorName === 'telegram') {
			return 'mdi:telegram';
		}
		if (connectorName === 'discord') {
			return 'mdi:discord';
		}
		return 'mdi:robot-outline';
	});

	async function fetchBot() {
		loading.value = true;
		error.value = '';
		try {
			const res = await window.client.getChatBots({});
			const list = res.bots || [];
			const routeIdentityValue = routeIdentityRaw.value === '_' ? '' : (routeIdentityRaw.value || '');
			bot.value = list.find(b => {
				const matchesConnector = b.connector === connector.value;
				const botIdentity = b.identity || '';
				return matchesConnector && botIdentity === routeIdentityValue;
			}) || null;
			if (!bot.value) {
				error.value = 'Bot not found.';
			}
		} catch (e) {
			error.value = `Failed to load bot: ${e.message || e}`;
		} finally {
			loading.value = false;
		}
	}

	onMounted(async () => {
		connector.value = route.params.connector ? decodeURIComponent(route.params.connector) : '';
		routeIdentityRaw.value = route.params.identity ? decodeURIComponent(route.params.identity) : '';
		await waitForClient();
		clientReady.value = true;
		await fetchBot();
	});
</script>

<style scoped>
	.bot-meta-row {
		display: inline-flex;
		align-items: center;
		gap: 0.4em;
		margin: 0 0 1em;
		color: var(--text-muted, #666);
	}

	.bot-meta-connector {
		font-weight: 600;
		color: var(--text-color, #1a1a1a);
	}

	.bot-meta-sep {
		opacity: 0.7;
	}

	@media (prefers-color-scheme: dark) {
		.bot-meta-row {
			color: var(--text-muted, #aaa);
		}

		.bot-meta-connector {
			color: var(--text-color, #f0f0f0);
		}
	}
</style>
