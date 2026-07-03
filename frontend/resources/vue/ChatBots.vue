<template>
	<Section
		title="Chat Bots"
		subtitle="Configured chat bots and their connection status."
		classes="chat-bots"
	>
		<template #toolbar>
			<router-link :to="{ name: 'createChatBot' }" class="button good">
				<Icon icon="mdi:plus" />
				Add chat bot
			</router-link>
		</template>

		<div v-if = "loading" class = "icon-and-text" style = "margin-top: 1em;">
			<Icon icon = "eos-icons:loading" width = "24" height = "24" />
			<span style = "margin-left: .5em;">Loading bots...</span>
		</div>

		<div v-else>
			<p v-if = "bots.length === 0" class = "inline-notification note">No chat bots configured. Use <router-link :to="{ name: 'createChatBot' }">Add chat bot</router-link> to create one.</p>

			<Navigation v-else ref = "localNavigation">
				<NavigationGrid />
			</Navigation>
		</div>
	</Section>
</template>

<script setup>
	import { ref, onMounted, nextTick, watch } from 'vue';
	import { Icon } from '@iconify/vue';
	import { useRouter } from 'vue-router';
	import { waitForClient } from '../javascript/util';
	import Section from 'picocrank/vue/components/Section.vue';
	import Navigation from 'picocrank/vue/components/Navigation.vue';
	import NavigationGrid from 'picocrank/vue/components/NavigationGrid.vue';
	import { Robot01Icon, TelegramIcon, DiscordIcon } from '@hugeicons/core-free-icons';

	const router = useRouter();
	const bots = ref([]);
	const loading = ref(true);
	const localNavigation = ref(null);
	const STOPPED_BOT_ICON_COLOR = '#dc3545';

	function getBotRoute(bot) {
		if (!bot) {
			return { name: 'chatBots' };
		}
		return {
			name: 'chatBotDetails',
			params: {
				protocol: String(bot.connector || ''),
				botId: String(bot.botId || ''),
			}
		};
	}

	function getBotDescription(bot) {
		const parts = [];
		if (bot.botId) {
			parts.push(`ID: ${bot.botId}`);
		}
		if (bot.identity) {
			parts.push(`Identity: ${bot.identity}`);
		}
		if (!bot.isRunning) {
			if (bot.statusMessage) {
				parts.push(bot.statusMessage);
			}
			if (bot.errorMessage) {
				parts.push(`Error: ${bot.errorMessage}`);
			}
		}
		return parts.join(' • ') || 'Chat bot';
	}

	function getBotIcon(bot) {
		const protocolIcons = {
			'telegram': TelegramIcon,
			'discord': DiscordIcon,
		};

		return protocolIcons[bot.connector?.toLowerCase()] || Robot01Icon;
	}

	function updateNavigation() {
		if (!localNavigation.value) {
			return;
		}

		localNavigation.value.clearNavigationLinks();

		for (const bot of bots.value) {
			const route = getBotRoute(bot);
			const statusText = bot.isRunning ? 'Running' : 'Stopped';
			const description = getBotDescription(bot);
			const fullDescription = `${statusText}${description ? ' • ' + description : ''}`;

			const displayName = bot.protocolDisplayName || bot.name;

			localNavigation.value.addCallback(displayName, () => {
				router.push(route);
			}, {
				icon: getBotIcon(bot),
				name: `bot-${bot.connector}-${bot.botId || ''}`,
				description: fullDescription,
				iconColor: bot.isRunning ? null : STOPPED_BOT_ICON_COLOR,
			});
		}
	}

	async function fetchBots() {
		await waitForClient();

		try {
			const response = await window.client.getChatBots({});
			bots.value = (response.bots || []).map(bot => ({
				...bot,
				connector: String(bot.connector || ''),
				botId: bot.botId ? String(bot.botId) : '',
				identity: bot.identity ? String(bot.identity) : '',
				name: String(bot.name || ''),
				protocolDisplayName: bot.protocolDisplayName ? String(bot.protocolDisplayName) : '',
				icon: bot.icon || 'mdi:robot',
				isRunning: Boolean(bot.isRunning),
				statusMessage: bot.statusMessage || '',
				errorMessage: bot.errorMessage || ''
			})).sort((a, b) => {
				if (a.connector !== b.connector) {
					return a.connector.localeCompare(b.connector);
				}
				if (a.botId !== b.botId) {
					return (a.botId || '').localeCompare(b.botId || '');
				}
				return (a.name || '').localeCompare(b.name || '');
			});
		} catch (e) {
			console.error('Failed to fetch chatbots:', e);
			bots.value = [];
		} finally {
			loading.value = false;
		}
	}

	watch([bots, localNavigation], async ([newBots, nav]) => {
		if (newBots.length > 0 && nav) {
			await nextTick();
			updateNavigation();
		}
	}, { immediate: false });

	onMounted(async () => {
		await fetchBots();
		await nextTick();

		if (localNavigation.value && bots.value.length > 0) {
			updateNavigation();
		} else {
			setTimeout(() => {
				if (localNavigation.value && bots.value.length > 0) {
					updateNavigation();
				}
			}, 300);
		}
	});
</script>

<style scoped>
</style>
