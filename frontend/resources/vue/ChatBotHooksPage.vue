<template>
	<Section
		title="Incoming message hooks"
		:subtitle="bot ? `${bot.name} · ${bot.connector}` : 'Webhook URLs for this bot'"
		classes="chat-bot-hooks-page"
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
			<p class="hooks-intro">
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
							<button type="button" @click="removeHook(index)" class="hook-remove-button">
								Remove
							</button>
						</div>
					</div>
				</div>

				<div class="add-hook-form">
					<h4 class="add-hook-heading">Add new hook</h4>
					<div class="add-hook-row">
						<input
							type="url"
							v-model="newHookUrl"
							placeholder="https://example.com/webhook"
							:disabled="savingHooks"
						/>
						<button type="button" @click="addHook" :disabled="!newHookUrl || savingHooks" class="good">
							Add hook
						</button>
					</div>
				</div>

				<div v-if="hooksMessage" class="inline-notification" :class="hooksMessageType" style="margin-top: 1em;">
					{{ hooksMessage }}
				</div>
			</div>
		</div>
	</Section>
</template>

<script setup>
	import { computed, ref, onMounted } from 'vue';
	import { useRoute } from 'vue-router';
	import { Icon } from '@iconify/vue';
	import Section from 'picocrank/vue/components/Section.vue';
	import { waitForClient } from '../javascript/util';

	const route = useRoute();
	const clientReady = ref(false);
	const loading = ref(true);
	const error = ref('');
	const connector = ref('');
	const routeIdentityRaw = ref('');
	const bot = ref(null);

	const hooks = ref([]);
	const hooksLoading = ref(false);
	const hooksError = ref('');
	const hooksMessage = ref('');
	const hooksMessageType = ref('');
	const savingHooks = ref(false);
	const newHookUrl = ref('');

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

	function routeIdentityValue() {
		return routeIdentityRaw.value === '_' ? '' : (routeIdentityRaw.value || '');
	}

	async function fetchBot() {
		loading.value = true;
		error.value = '';
		try {
			const res = await window.client.getChatBots({});
			const list = res.bots || [];
			const idVal = routeIdentityValue();
			bot.value = list.find(b => {
				const matchesConnector = b.connector === connector.value;
				const botIdentity = b.identity || '';
				return matchesConnector && botIdentity === idVal;
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

	async function fetchHooks() {
		if (!bot.value) {
			return;
		}
		hooksLoading.value = true;
		hooksError.value = '';
		hooksMessage.value = '';
		try {
			const res = await window.client.getBotHooks({
				connector: bot.value.connector,
				identity: routeIdentityValue(),
			});
			hooks.value = (res.hooks || []).map(hook => ({
				url: String(hook.url || ''),
				enabled: Boolean(hook.enabled),
			}));
		} catch (e) {
			hooksError.value = `Failed to load hooks: ${e.message || e}`;
			hooks.value = [];
		} finally {
			hooksLoading.value = false;
		}
	}

	async function saveHooks() {
		if (!bot.value || savingHooks.value) {
			return;
		}
		savingHooks.value = true;
		hooksMessage.value = '';
		hooksMessageType.value = '';
		try {
			const res = await window.client.setBotHooks({
				connector: bot.value.connector,
				identity: routeIdentityValue(),
				hooks: hooks.value.map(hook => ({
					url: hook.url,
					enabled: hook.enabled,
				})),
			});
			if (res.standardResponse.success) {
				hooksMessage.value = 'Hooks saved successfully';
				hooksMessageType.value = 'note';
			} else {
				hooksMessage.value = res.standardResponse.errorMessage || 'Failed to save hooks';
				hooksMessageType.value = 'error';
			}
		} catch (e) {
			hooksMessage.value = `Failed to save hooks: ${e.message || e}`;
			hooksMessageType.value = 'error';
		} finally {
			savingHooks.value = false;
		}
	}

	function addHook() {
		if (!newHookUrl.value.trim()) {
			return;
		}
		hooks.value.push({
			url: newHookUrl.value.trim(),
			enabled: true,
		});
		newHookUrl.value = '';
		saveHooks();
	}

	function removeHook(index) {
		hooks.value.splice(index, 1);
		saveHooks();
	}

	onMounted(async () => {
		connector.value = route.params.connector ? decodeURIComponent(route.params.connector) : '';
		routeIdentityRaw.value = route.params.identity ? decodeURIComponent(route.params.identity) : '';
		await waitForClient();
		clientReady.value = true;
		await fetchBot();
		if (bot.value) {
			await fetchHooks();
		}
	});
</script>

<style scoped>
	.hooks-intro {
		margin: 0 0 1em;
		color: var(--text-muted, #666);
		line-height: 1.45;
	}

	.hook-item {
		padding: 1em;
		background: var(--conversations-item-bg, #f8f9fa);
		border-radius: 0.25em;
		margin-bottom: 0.75em;
		border: 1px solid var(--conversations-border, #e0e0e0);
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
		color: var(--text-muted, #666);
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
		background: var(--input-bg, #fff);
		border: 1px solid var(--border-color, #ccc);
		border-radius: 0.25em;
		color: var(--text-color, #1a1a1a);
		font-size: 0.9em;
		font-family: ui-monospace, 'Courier New', monospace;
		word-break: break-all;
		overflow-wrap: break-word;
		white-space: normal;
		line-height: 1.4;
	}

	.hook-remove-button {
		cursor: pointer;
		border: 1px solid var(--conversations-border, #ccc);
		border-radius: 0.25em;
		font-size: 0.9em;
		padding: 0.5em 1em;
		white-space: nowrap;
		background: var(--conversations-item-bg, #f0f0f0);
		color: var(--text-color, #1a1a1a);
	}

	.add-hook-form {
		margin-top: 1.5em;
		padding-top: 1.5em;
		border-top: 1px solid var(--conversations-border, #e0e0e0);
	}

	.add-hook-heading {
		margin: 0 0 0.75em;
		font-size: 1rem;
	}

	.add-hook-row {
		display: flex;
		gap: 0.5em;
		align-items: flex-start;
	}

	.add-hook-row input {
		flex: 1;
		padding: 0.5em;
		background: var(--input-bg, #fff);
		border: 1px solid var(--border-color, #ccc);
		border-radius: 0.25em;
		color: var(--text-color, #1a1a1a);
		font-size: 1em;
		box-sizing: border-box;
	}

	.add-hook-row input:focus {
		outline: none;
		border-color: var(--primary-color, #1976d2);
	}

	.add-hook-row button:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}

	@media (prefers-color-scheme: dark) {
		.hooks-intro {
			color: var(--text-muted, #aaa);
		}

		.hook-item {
			background: #2a2a2a;
			border-color: #333;
		}

		.hook-status-label {
			color: #aaa;
		}

		.hook-url {
			background: #1a1a1a;
			border-color: #444;
			color: #fff;
		}

		.hook-remove-button {
			background: #333;
			border-color: #444;
			color: #fff;
		}

		.add-hook-form {
			border-top-color: #444;
		}

		.add-hook-row input {
			background: #1a1a1a;
			border-color: #444;
			color: #fff;
		}

		.add-hook-row input:focus {
			border-color: #666;
		}
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
