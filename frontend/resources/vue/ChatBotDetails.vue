<template>
	<Section
		:title="bot?.name || 'Chat Bot'"
		:subtitle="bot ? bot.connector : 'Loading bot details'"
		classes="chat-bot-details"
	>
		<template #toolbar>
			<router-link :to="{ name: 'chatBots' }" class="button neutral">
				<Icon icon="mdi:robot-outline" />
				All chat bots
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
			<dl>
				<dt>Name</dt>
				<dd>{{ bot.name }}</dd>
				<dt>Connector</dt>
				<dd class="connector-with-icon">
					<Icon :icon="protocolIcon" width="18" height="18" />
					<span>{{ bot.connector }}</span>
				</dd>
				<dt>Identity</dt>
				<dd>{{ bot.identity || 'N/A' }}</dd>
				<dt>Status</dt>
				<dd>
					<div class="tag" :class="bot.isRunning ? 'fg-good' : 'fg-bad'">
						{{ bot.isRunning ? 'Running' : 'Stopped' }}
					</div>
				</dd>
				<dt v-if="bot.statusMessage">Status Message</dt>
				<dd v-if="bot.statusMessage">
					<div class="status-message" :class="bot.isRunning ? 'note' : 'warning'">
						{{ bot.statusMessage }}
					</div>
				</dd>
				<dt v-if="bot.errorMessage">Error</dt>
				<dd v-if="bot.errorMessage">
					<div class="inline-notification error">
						{{ bot.errorMessage }}
					</div>
				</dd>
			</dl>

			<div class="bot-detail-nav-wrap">
				<Navigation ref="botDetailNav">
					<NavigationGrid />
				</Navigation>
			</div>

			<!-- Channels Section -->
			<div v-if="bot.connector === 'telegram'" id="chat-bot-channels" class="channels-section" style="margin-top: 2em;">
				<h3>Channels</h3>
				<div v-if="channelsLoading" class="icon-and-text" style="margin-top: 1em;">
					<Icon icon="eos-icons:loading" width="24" height="24" />
					<span style="margin-left: .5em;">Loading channels...</span>
				</div>
				<div v-else-if="channelsError" class="inline-notification error">
					{{ channelsError }}
				</div>
				<div v-else-if="channels.length === 0" class="inline-notification note">
					No channels found. Channels will appear here once the bot receives messages from them.
				</div>
				<table v-else class="data-table" style="margin-top: 1em;">
					<thead>
						<tr>
							<th>Title</th>
							<th>Type</th>
							<th>Username</th>
							<th>ID</th>
						</tr>
					</thead>
					<tbody>
						<tr v-for="channel in channels" :key="channel.id">
							<td>{{ channel.title }}</td>
							<td>
								<div class="tag fg-neutral">{{ channel.type }}</div>
							</td>
							<td>{{ channel.username || 'N/A' }}</td>
							<td><code>{{ channel.id }}</code></td>
						</tr>
					</tbody>
				</table>
			</div>
		</div>
	</Section>
</template>

<script setup>
	import { ref, computed, onMounted, watch, nextTick } from 'vue';
	import { useRoute, useRouter } from 'vue-router';
	import { Icon } from '@iconify/vue';
	import Section from 'picocrank/vue/components/Section.vue';
	import Navigation from 'picocrank/vue/components/Navigation.vue';
	import NavigationGrid from 'picocrank/vue/components/NavigationGrid.vue';
	import {
		MessageMultiple01Icon,
		WebhookIcon,
	} from '@hugeicons/core-free-icons';

	const route = useRoute();
	const router = useRouter();

	const clientReady = ref(false)
	const loading = ref(true)
	const error = ref('')
	const connector = ref('')
	const identity = ref('')
	const bot = ref(null)
	const channels = ref([])
	const channelsLoading = ref(false)
	const channelsError = ref('')
	const botDetailNav = ref(null)

	const identityForRoute = computed(() => identity.value || '_')
	const protocolIcon = computed(() => {
		const connectorName = String(bot.value?.connector || '').toLowerCase()
		if (connectorName === 'telegram') {
			return 'mdi:telegram'
		}
		if (connectorName === 'discord') {
			return 'mdi:discord'
		}
		return 'mdi:robot-outline'
	})

	function goToConversations() {
		if (!bot.value) {
			return
		}
		router.push({
			name: 'chatBotConversations',
			params: { connector: bot.value.connector, identity: identityForRoute.value },
		})
	}

	function goToMessageHooks() {
		if (!bot.value) {
			return
		}
		router.push({
			name: 'chatBotHooks',
			params: { connector: bot.value.connector, identity: identityForRoute.value },
		})
	}

	async function rebuildBotDetailNavigation() {
		await nextTick()
		const nav = botDetailNav.value
		if (!nav || !bot.value) {
			return
		}
		nav.clearNavigationLinks()
		nav.addCallback('Conversations', goToConversations, {
			icon: MessageMultiple01Icon,
			name: 'bot-detail-conversations',
			description: 'Message threads and replies for this bot',
		})
		nav.addCallback('Message hooks', goToMessageHooks, {
			icon: WebhookIcon,
			name: 'bot-detail-hooks',
			description: 'Configure incoming webhooks for this bot',
		})
	}

	watch(
		bot,
		(b) => {
			if (b) {
				rebuildBotDetailNavigation()
			}
		},
		{ flush: 'post' },
	)

	function waitForClient() {
		return new Promise((resolve) => {
			const check = () => {
				if (window.client) resolve(true); else setTimeout(check, 100);
			};
			check();
		});
	}

	async function fetchBot() {
		loading.value = true
		error.value = ''
		try {
			const res = await window.client.getChatBots({})
			const list = res.bots || []
			// Handle placeholder value for empty identities
			const routeIdentityValue = identity.value === '_' ? '' : (identity.value || '')
			bot.value = list.find(b => {
				const matchesConnector = b.connector === connector.value
				const botIdentity = b.identity || ''
				// Match if connector matches and identity matches (both empty or both same value)
				return matchesConnector && botIdentity === routeIdentityValue
			}) || null
			if (!bot.value) {
				error.value = 'Bot not found.'
			}
		} catch (e) {
			error.value = `Failed to load bot: ${e.message || e}`
		} finally {
			loading.value = false
		}
	}

	async function fetchChannels() {
		if (!bot.value || bot.value.connector !== 'telegram') {
			return
		}

		channelsLoading.value = true
		channelsError.value = ''
		try {
			const routeIdentityValue = identity.value === '_' ? '' : (identity.value || '')
			const res = await window.client.getBotChannels({
				connector: bot.value.connector,
				identity: routeIdentityValue
			})
			channels.value = (res.channels || []).map(ch => ({
				id: String(ch.id || ''),
				title: String(ch.title || ''),
				type: String(ch.type || ''),
				username: String(ch.username || '')
			}))
		} catch (e) {
			channelsError.value = `Failed to load channels: ${e.message || e}`
			channels.value = []
		} finally {
			channelsLoading.value = false
		}
	}

	onMounted(async () => {
		connector.value = route.params.connector ? decodeURIComponent(route.params.connector) : ''
		identity.value = route.params.identity ? decodeURIComponent(route.params.identity) : ''
		await waitForClient()
		clientReady.value = true
		await fetchBot()
		if (bot.value) {
			await fetchChannels()
		}
	})
</script>

<style scoped>
	.text-disabled {
		opacity: 0.5;
		text-decoration: line-through;
	}

	.bot-detail-nav-wrap {
		margin-top: 1.5em;
	}

	.connector-with-icon {
		display: inline-flex;
		align-items: center;
		gap: 0.4em;
	}
</style>
