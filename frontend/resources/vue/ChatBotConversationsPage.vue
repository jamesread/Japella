<template>
	<Section
		title="Conversations"
		:subtitle="bot ? `${bot.name} · ${bot.connector} · ${bot.botId}` : 'Chat bot messages'"
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
			<div class="bot-meta">
				<Icon :icon="protocolIcon" width="18" height="18" />
				<span class="bot-meta-connector">{{ bot.connector }}</span>
				<span class="bot-meta-sep">·</span>
				<span class="bot-meta-id">ID: {{ bot.botId }}</span>
				<span v-if="bot.identity" class="bot-meta-sep">·</span>
				<span v-if="bot.identity" class="bot-meta-identity">Identity: {{ bot.identity }}</span>
			</div>
			<ChatBotConversations :bot="bot" />
		</div>
	</Section>
</template>

<script setup>
	import { computed, ref, onMounted } from 'vue';
	import { useRoute } from 'vue-router';
	import { Icon } from '@iconify/vue';
	import Section from 'picocrank/vue/components/Section.vue';
	import ChatBotConversations from './ChatBotConversations.vue';
	import { waitForClient } from '../javascript/util';

	const route = useRoute();
	const clientReady = ref(false);
	const loading = ref(true);
	const error = ref('');
	const protocol = ref('');
	const botId = ref('');
	const bot = ref(null);

	const detailsParams = computed(() => ({
		protocol: route.params.protocol
			? decodeURIComponent(String(route.params.protocol))
			: protocol.value,
		botId: route.params.botId
			? decodeURIComponent(String(route.params.botId))
			: botId.value,
	}));

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
			bot.value = list.find(b => {
				return b.connector === protocol.value && String(b.botId || '') === botId.value;
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
		protocol.value = route.params.protocol ? decodeURIComponent(route.params.protocol) : '';
		botId.value = route.params.botId ? decodeURIComponent(route.params.botId) : '';
		await waitForClient();
		clientReady.value = true;
		await fetchBot();
	});
</script>

<style scoped>
	.bot-meta {
		display: flex;
		align-items: center;
		gap: 0.35em;
		margin-bottom: 1em;
		color: var(--text-muted, #666);
		flex-wrap: wrap;
	}

	.bot-meta-connector {
		font-weight: 500;
		color: var(--text-color, #1a1a1a);
	}

	.bot-meta-sep {
		opacity: 0.6;
	}

	@media (max-width: 480px) {
		.bot-meta-connector {
			width: 100%;
		}
	}
</style>
