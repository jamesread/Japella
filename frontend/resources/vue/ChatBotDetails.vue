<template>
	<Section
		:title="bot?.name || 'Chat Bot'"
		:subtitle="bot ? bot.connector : 'Loading bot details'"
		classes="chat-bot-details"
	>
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
				<dd>{{ bot.connector }}</dd>
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

			<!-- Channels Section -->
			<div v-if="bot.connector === 'telegram'" class="channels-section" style="margin-top: 2em;">
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

			<!-- Hooks Section -->
			<div class="hooks-section" style="margin-top: 2em;">
				<h3>Incoming Message Hooks</h3>
				<p style="margin-top: 0.5em; margin-bottom: 1em; color: #aaa;">
					Configure webhooks to be called when this bot receives messages. Each hook will receive a JSON payload with message details.
				</p>

				<div v-if="hooksLoading" class="icon-and-text" style="margin-top: 1em;">
					<Icon icon="eos-icons:loading" width="20" height="20" />
					<span style="margin-left: .5em;">Loading hooks...</span>
				</div>
				<div v-else-if="hooksError" class="inline-notification error">
					{{ hooksError }}
				</div>
				<div v-else>
					<div v-if="hooks.length === 0" class="inline-notification note">
						No hooks configured. Add a webhook URL below to start receiving message notifications.
					</div>
					<div v-else class="hooks-list" style="margin-top: 1em;">
						<div v-for="(hook, index) in hooks" :key="index" class="hook-item">
							<div class="hook-item-content">
								<label class="hook-checkbox-label">
									<input type="checkbox" v-model="hook.enabled" @change="saveHooks" />
									<span class="hook-status-label">{{ hook.enabled ? 'Enabled' : 'Disabled' }}</span>
								</label>
								<div class="hook-url-container" :class="{ 'hook-disabled': !hook.enabled }">
									<code class="hook-url">{{ hook.url }}</code>
								</div>
								<button @click="removeHook(index)" class="hook-remove-button">Remove</button>
							</div>
						</div>
					</div>

					<div class="add-hook-form" style="margin-top: 1.5em; padding-top: 1.5em; border-top: 1px solid #444;">
						<h4 style="margin-top: 0; margin-bottom: 0.75em;">Add New Hook</h4>
						<div style="display: flex; gap: 0.5em; align-items: flex-start;">
							<input
								type="url"
								v-model="newHookUrl"
								placeholder="https://example.com/webhook"
								style="flex: 1; padding: 0.5em;"
								:disabled="savingHooks"
							/>
							<button @click="addHook" :disabled="!newHookUrl || savingHooks" class="good" style="padding: 0.5em 1em;">
								Add Hook
							</button>
						</div>
					</div>

					<div v-if="hooksMessage" class="inline-notification" :class="hooksMessageType" style="margin-top: 1em;">
						{{ hooksMessage }}
					</div>
				</div>
			</div>
		</div>
	</Section>
</template>

<script setup>
	import { ref, onMounted } from 'vue';
	import { useRoute } from 'vue-router';
	import { Icon } from '@iconify/vue';
	import Section from 'picocrank/vue/components/Section.vue';

	const route = useRoute();
	const clientReady = ref(false)
	const loading = ref(true)
	const error = ref('')
	const connector = ref('')
	const identity = ref('')
	const bot = ref(null)
	const channels = ref([])
	const channelsLoading = ref(false)
	const channelsError = ref('')
	const hooks = ref([])
	const hooksLoading = ref(false)
	const hooksError = ref('')
	const hooksMessage = ref('')
	const hooksMessageType = ref('')
	const savingHooks = ref(false)
	const newHookUrl = ref('')

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

	async function fetchHooks() {
		if (!bot.value) {
			return
		}

		hooksLoading.value = true
		hooksError.value = ''
		hooksMessage.value = ''
		try {
			const routeIdentityValue = identity.value === '_' ? '' : (identity.value || '')
			const res = await window.client.getBotHooks({
				connector: bot.value.connector,
				identity: routeIdentityValue
			})
			hooks.value = (res.hooks || []).map(hook => ({
				url: String(hook.url || ''),
				enabled: Boolean(hook.enabled)
			}))
		} catch (e) {
			hooksError.value = `Failed to load hooks: ${e.message || e}`
			hooks.value = []
		} finally {
			hooksLoading.value = false
		}
	}

	async function saveHooks() {
		if (!bot.value || savingHooks.value) {
			return
		}

		savingHooks.value = true
		hooksMessage.value = ''
		hooksMessageType.value = ''
		try {
			const routeIdentityValue = identity.value === '_' ? '' : (identity.value || '')
			const res = await window.client.setBotHooks({
				connector: bot.value.connector,
				identity: routeIdentityValue,
				hooks: hooks.value.map(hook => ({
					url: hook.url,
					enabled: hook.enabled
				}))
			})
			if (res.standardResponse.success) {
				hooksMessage.value = 'Hooks saved successfully'
				hooksMessageType.value = 'note'
			} else {
				hooksMessage.value = res.standardResponse.errorMessage || 'Failed to save hooks'
				hooksMessageType.value = 'error'
			}
		} catch (e) {
			hooksMessage.value = `Failed to save hooks: ${e.message || e}`
			hooksMessageType.value = 'error'
		} finally {
			savingHooks.value = false
		}
	}

	function addHook() {
		if (!newHookUrl.value.trim()) {
			return
		}
		hooks.value.push({
			url: newHookUrl.value.trim(),
			enabled: true
		})
		newHookUrl.value = ''
		saveHooks()
	}

	function removeHook(index) {
		hooks.value.splice(index, 1)
		saveHooks()
	}

	onMounted(async () => {
		connector.value = route.params.connector ? decodeURIComponent(route.params.connector) : ''
		identity.value = route.params.identity ? decodeURIComponent(route.params.identity) : ''
		await waitForClient()
		clientReady.value = true
		await fetchBot()
		if (bot.value) {
			await Promise.all([fetchChannels(), fetchHooks()])
		}
	})
</script>

<style scoped>
	.text-disabled {
		opacity: 0.5;
		text-decoration: line-through;
	}

	.hook-item {
		padding: 1em;
		background: #2a2a2a;
		border-radius: 0.25em;
		margin-bottom: 0.75em;
		border: 1px solid #333;
	}

	.hook-item-content {
		display: grid;
		grid-template-columns: auto 1fr auto;
		gap: 1em;
		align-items: center;
	}

	.hook-checkbox-label {
		display: flex;
		align-items: center;
		gap: 0.5em;
		cursor: pointer;
		white-space: nowrap;
	}

	.hook-checkbox-label input[type="checkbox"] {
		cursor: pointer;
		margin: 0;
	}

	.hook-status-label {
		font-size: 0.9em;
		color: #aaa;
		min-width: 60px;
	}

	.hook-url-container {
		flex: 1;
		min-width: 0;
		overflow: hidden;
	}

	.hook-url-container.hook-disabled {
		opacity: 0.5;
	}

	.hook-url {
		display: block;
		padding: 0.5em 0.75em;
		background: #1a1a1a;
		border: 1px solid #444;
		border-radius: 0.25em;
		color: #fff;
		font-size: 0.9em;
		font-family: 'Courier New', monospace;
		word-break: break-all;
		overflow-wrap: break-word;
		white-space: normal;
		line-height: 1.4;
	}

	.hook-remove-button {
		cursor: pointer;
		border: none;
		border-radius: 0.25em;
		font-size: 0.9em;
		padding: 0.5em 1em;
		white-space: nowrap;
	}

	.add-hook-form input {
		background: #1a1a1a;
		border: 1px solid #444;
		border-radius: 0.25em;
		color: #fff;
		font-size: 1em;
	}

	.add-hook-form input:focus {
		outline: none;
		border-color: #666;
	}

	.add-hook-form button:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}

	@media (max-width: 768px) {
		.hook-item-content {
			grid-template-columns: 1fr;
			gap: 0.75em;
		}

		.hook-checkbox-label {
			order: 1;
		}

		.hook-url-container {
			order: 2;
		}

		.hook-remove-button {
			order: 3;
			width: 100%;
		}
	}
</style>
