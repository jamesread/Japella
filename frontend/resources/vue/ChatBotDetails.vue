<template>
	<Section
		:title="bot?.name || 'Chat Bot'"
		:subtitle="bot ? `${bot.connector} · ${bot.botId}` : 'Loading bot details'"
		classes="chat-bot-details"
	>
		<template #toolbar>
			<button
				v-if="bot && !bot.isRunning"
				type="button"
				class="button good"
				:disabled="actionLoading"
				@click="startBot"
			>
				<Icon icon="mdi:play" />
				Start bot
			</button>
			<button
				v-if="bot && bot.isRunning"
				type="button"
				class="button warning"
				:disabled="actionLoading"
				@click="stopBot"
			>
				<Icon icon="mdi:stop" />
				Stop bot
			</button>
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
				<dt>Bot ID</dt>
				<dd><code>{{ bot.botId }}</code></dd>
				<dt>Display name</dt>
				<dd>{{ bot.name }}</dd>
				<dt>Connector</dt>
				<dd class="connector-with-icon">
					<Icon :icon="protocolIcon" width="18" height="18" />
					<span>{{ bot.connector }}</span>
				</dd>
				<dt>Platform identity</dt>
				<dd>{{ bot.identity || 'N/A (start the bot to connect)' }}</dd>
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

			<div class="edit-bot-section">
				<h3>Settings</h3>
				<form class="edit-bot-form" @submit.prevent="saveBot">
					<label for="edit-display-name">Display name</label>
					<input id="edit-display-name" v-model="editForm.displayName" type="text" :disabled="actionLoading" />

					<template v-if="bot.connector === 'telegram'">
						<label for="edit-telegram-token">Bot token</label>
						<input
							id="edit-telegram-token"
							v-model="editForm.telegramBotToken"
							type="password"
							autocomplete="off"
							placeholder="Leave blank to keep current token"
							:disabled="actionLoading"
						/>
					</template>

					<template v-else-if="bot.connector === 'discord'">
						<label for="edit-discord-token">Bot token</label>
						<input
							id="edit-discord-token"
							v-model="editForm.discordToken"
							type="password"
							autocomplete="off"
							placeholder="Leave blank to keep current token"
							:disabled="actionLoading"
						/>
						<label for="edit-discord-app-id">Application ID</label>
						<input id="edit-discord-app-id" v-model="editForm.discordAppId" type="text" :disabled="actionLoading" />
						<label for="edit-discord-public-key">Public key</label>
						<input id="edit-discord-public-key" v-model="editForm.discordPublicKey" type="text" :disabled="actionLoading" />
					</template>

					<div class="edit-bot-actions">
						<button type="submit" class="good" :disabled="actionLoading">Save changes</button>
						<button type="button" class="warning" :disabled="actionLoading" @click="deleteBot">Delete bot</button>
					</div>
					<p v-if="editMessage" class="inline-notification" :class="editMessageType">{{ editMessage }}</p>
				</form>
			</div>

			<div class="bot-detail-nav-wrap">
				<Navigation ref="botDetailNav">
					<NavigationGrid />
				</Navigation>
			</div>

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
	const protocol = ref('')
	const botId = ref('')
	const bot = ref(null)
	const channels = ref([])
	const channelsLoading = ref(false)
	const channelsError = ref('')
	const botDetailNav = ref(null)
	const actionLoading = ref(false)
	const editForm = ref({
		displayName: '',
		telegramBotToken: '',
		discordToken: '',
		discordAppId: '',
		discordPublicKey: '',
	})
	const editMessage = ref('')
	const editMessageType = ref('')

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

	function botApiParams() {
		return {
			protocol: bot.value.connector,
			botId: bot.value.botId,
		}
	}

	function goToConversations() {
		if (!bot.value) {
			return
		}
		router.push({
			name: 'chatBotConversations',
			params: { protocol: bot.value.connector, botId: bot.value.botId },
		})
	}

	function goToMessageHooks() {
		if (!bot.value) {
			return
		}
		router.push({
			name: 'chatBotHooks',
			params: { protocol: bot.value.connector, botId: bot.value.botId },
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
				editForm.value.displayName = b.name || ''
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
			bot.value = list.find(b => {
				return b.connector === protocol.value && String(b.botId || '') === botId.value
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

	async function startBot() {
		if (!bot.value || actionLoading.value) {
			return
		}
		actionLoading.value = true
		error.value = ''
		try {
			const res = await window.client.startChatBot(botApiParams())
			if (res.bot) {
				bot.value = res.bot
			} else {
				await fetchBot()
			}
			if (bot.value?.connector === 'telegram') {
				await fetchChannels()
			}
		} catch (e) {
			error.value = `Failed to start bot: ${e.message || e}`
		} finally {
			actionLoading.value = false
		}
	}

	async function stopBot() {
		if (!bot.value || actionLoading.value) {
			return
		}
		actionLoading.value = true
		error.value = ''
		try {
			const res = await window.client.stopChatBot(botApiParams())
			if (res.bot) {
				bot.value = res.bot
			} else {
				await fetchBot()
			}
		} catch (e) {
			error.value = `Failed to stop bot: ${e.message || e}`
		} finally {
			actionLoading.value = false
		}
	}

	async function saveBot() {
		if (!bot.value || actionLoading.value) {
			return
		}
		actionLoading.value = true
		editMessage.value = ''
		try {
			const req = {
				protocol: bot.value.connector,
				botId: bot.value.botId,
				displayName: editForm.value.displayName.trim(),
			}
			if (bot.value.connector === 'telegram' && editForm.value.telegramBotToken.trim()) {
				req.telegramBotToken = editForm.value.telegramBotToken.trim()
			}
			if (bot.value.connector === 'discord') {
				if (editForm.value.discordToken.trim()) {
					req.discordToken = editForm.value.discordToken.trim()
				}
				if (editForm.value.discordAppId.trim()) {
					req.discordAppId = editForm.value.discordAppId.trim()
				}
				if (editForm.value.discordPublicKey.trim()) {
					req.discordPublicKey = editForm.value.discordPublicKey.trim()
				}
			}
			const res = await window.client.updateChatBot(req)
			if (res.bot) {
				bot.value = res.bot
			} else {
				await fetchBot()
			}
			editForm.value.telegramBotToken = ''
			editForm.value.discordToken = ''
			editMessage.value = 'Bot updated'
			editMessageType.value = 'note'
		} catch (e) {
			editMessage.value = `Failed to update bot: ${e.message || e}`
			editMessageType.value = 'error'
		} finally {
			actionLoading.value = false
		}
	}

	async function deleteBot() {
		if (!bot.value || actionLoading.value) {
			return
		}
		if (!window.confirm(`Delete chat bot "${bot.value.name}" (${bot.value.botId})? This cannot be undone.`)) {
			return
		}
		actionLoading.value = true
		try {
			await window.client.deleteChatBot({
				protocol: bot.value.connector,
				botId: bot.value.botId,
			})
			await router.push({ name: 'chatBots' })
		} catch (e) {
			editMessage.value = `Failed to delete bot: ${e.message || e}`
			editMessageType.value = 'error'
			actionLoading.value = false
		}
	}

	async function fetchChannels() {
		if (!bot.value || bot.value.connector !== 'telegram') {
			return
		}

		channelsLoading.value = true
		channelsError.value = ''
		try {
			const res = await window.client.getBotChannels(botApiParams())
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
		protocol.value = route.params.protocol ? decodeURIComponent(route.params.protocol) : ''
		botId.value = route.params.botId ? decodeURIComponent(route.params.botId) : ''
		await waitForClient()
		clientReady.value = true
		await fetchBot()
		if (bot.value) {
			await fetchChannels()
		}
	})
</script>

<style scoped>
	.bot-detail-nav-wrap {
		margin-top: 1.5em;
	}

	.connector-with-icon {
		display: inline-flex;
		align-items: center;
		gap: 0.4em;
	}

	.edit-bot-section {
		margin-top: 2em;
		padding-top: 1.5em;
		border-top: 1px solid var(--conversations-border, #e0e0e0);
	}

	.edit-bot-form {
		display: grid;
		grid-template-columns: minmax(8rem, 12rem) 1fr;
		gap: 0.75rem 1rem;
		max-width: 40rem;
		align-items: start;
	}

	.edit-bot-form label {
		padding-top: 0.35rem;
	}

	.edit-bot-form input {
		width: 100%;
		box-sizing: border-box;
	}

	.edit-bot-actions {
		grid-column: 1 / -1;
		display: flex;
		gap: 0.5rem;
		flex-wrap: wrap;
		margin-top: 0.5rem;
	}
</style>
