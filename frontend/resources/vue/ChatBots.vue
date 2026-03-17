<template>
	<Section
		title="Chat Bots"
		subtitle="These bots are currently configured and running."
		classes="chat-bots"
	>
		<div v-if = "loading" class = "icon-and-text" style = "margin-top: 1em;">
			<Icon icon = "eos-icons:loading" width = "24" height = "24" />
			<span style = "margin-left: .5em;">Loading bots...</span>
		</div>

		<div v-else>
			<p v-if = "bots.length === 0" class = "inline-notification note">No bots running.</p>

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

	function getBotRoute(bot) {
		if (!bot) {
			return { name: 'chatBots' };
		}
		// Use a placeholder for empty identities to avoid Vue Router issues
		const identityValue = bot.identity && bot.identity.trim() ? String(bot.identity) : '_';
		return {
			name: 'chatBotDetails',
			params: {
				connector: String(bot.connector || ''),
				identity: identityValue
			}
		};
	}

	function getBotDescription(bot) {
		const parts = [];
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
		// Static lookup for ChatBot icons based on protocol
		const protocolIcons = {
			'telegram': TelegramIcon,
			'discord': DiscordIcon,
			// Add more protocol icons here as needed
		};

		// Return protocol-specific icon if available, otherwise use default Robot01Icon
		return protocolIcons[bot.connector?.toLowerCase()] || Robot01Icon;
	}

	function updateNavigation() {
		if (!localNavigation.value) {
			return;
		}

		// Add each bot as a navigation item using addCallback
		for (const bot of bots.value) {
			const route = getBotRoute(bot);
			const statusText = bot.isRunning ? 'Running' : 'Stopped';
			const description = getBotDescription(bot);
			const fullDescription = `${statusText}${description ? ' • ' + description : ''}`;

			// Use ProtocolDisplayName if available, otherwise fall back to name
			const displayName = bot.protocolDisplayName || bot.name;

			// Use addCallback with a navigation function, similar to ControlPanel
			localNavigation.value.addCallback(displayName, () => {
				router.push(route);
			}, {
				icon: getBotIcon(bot), // Use protocol-specific icon
				name: `bot-${bot.connector}-${bot.identity || ''}`,
				description: fullDescription
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
				identity: bot.identity ? String(bot.identity) : '',
				name: String(bot.name || ''),
				protocolDisplayName: bot.protocolDisplayName ? String(bot.protocolDisplayName) : '',
				icon: bot.icon || 'mdi:robot',
				isRunning: Boolean(bot.isRunning),
				statusMessage: bot.statusMessage || '',
				errorMessage: bot.errorMessage || ''
			})).sort((a, b) => {
				// Sort deterministically: first by connector, then by identity, then by name
				if (a.connector !== b.connector) {
					return a.connector.localeCompare(b.connector);
				}
				if (a.identity !== b.identity) {
					return (a.identity || '').localeCompare(b.identity || '');
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

	// Watch for when bots are loaded and Navigation is ready
	watch([bots, localNavigation], async ([newBots, nav]) => {
		if (newBots.length > 0 && nav) {
			await nextTick();
			updateNavigation();
		}
	}, { immediate: false });

	onMounted(async () => {
		await fetchBots();
		// Wait for Navigation component to be ready after bots are loaded
		await nextTick();

		// Try to update navigation
		if (localNavigation.value && bots.value.length > 0) {
			updateNavigation();
		} else {
			// If not ready yet, wait a bit more
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
