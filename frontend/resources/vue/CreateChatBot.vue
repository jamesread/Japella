<template>
	<Section
		title="Add chat bot"
		subtitle="Create a Telegram or Discord bot instance. The bot ID cannot be changed later."
		classes="create-chat-bot"
	>
		<template #toolbar>
			<router-link :to="{ name: 'chatBots' }" class="button neutral">
				<Icon icon="material-symbols:arrow-back" />
				All chat bots
			</router-link>
		</template>

		<form class="create-bot-panel" @submit.prevent="createBot">
			<div class="create-bot-grid">
				<label for="bot-protocol">Protocol</label>
				<select id="bot-protocol" v-model="form.protocol" :disabled="creating" required>
					<option value="telegram">Telegram</option>
					<option value="discord">Discord</option>
				</select>

				<label for="bot-id">Bot ID</label>
				<div>
					<input
						id="bot-id"
						v-model="form.botId"
						type="text"
						pattern="[a-z][a-z0-9_-]{1,30}"
						autocomplete="off"
						required
						:disabled="creating"
						placeholder="e.g. support-bot"
					/>
					<small class="field-hint">Lowercase letters, numbers, hyphens, underscores (2–31 chars). Immutable after creation.</small>
				</div>

				<label for="bot-display-name">Display name</label>
				<input
					id="bot-display-name"
					v-model="form.displayName"
					type="text"
					autocomplete="off"
					:disabled="creating"
					placeholder="Optional friendly name"
				/>

				<template v-if="form.protocol === 'telegram'">
					<label for="telegram-token">Bot token</label>
					<input
						id="telegram-token"
						v-model="form.telegramBotToken"
						type="password"
						autocomplete="off"
						required
						:disabled="creating"
						placeholder="From @BotFather"
					/>
				</template>

				<template v-else-if="form.protocol === 'discord'">
					<label for="discord-token">Bot token</label>
					<input
						id="discord-token"
						v-model="form.discordToken"
						type="password"
						autocomplete="off"
						required
						:disabled="creating"
					/>

					<label for="discord-app-id">Application ID</label>
					<input
						id="discord-app-id"
						v-model="form.discordAppId"
						type="text"
						autocomplete="off"
						:disabled="creating"
					/>

					<label for="discord-public-key">Public key</label>
					<input
						id="discord-public-key"
						v-model="form.discordPublicKey"
						type="text"
						autocomplete="off"
						:disabled="creating"
					/>
				</template>

				<div class="create-bot-actions span-all">
					<button type="submit" class="good" :disabled="creating || !isFormValid">
						<Icon v-if="creating" icon="eos-icons:loading" width="16" height="16" />
						<Icon v-else icon="mdi:robot-outline" width="16" height="16" />
						<span>{{ creating ? 'Creating…' : 'Create chat bot' }}</span>
					</button>
				</div>

				<div v-if="message" class="span-all">
					<p class="inline-notification" :class="messageType">{{ message }}</p>
				</div>
			</div>
		</form>
	</Section>
</template>

<script setup>
	import { ref, computed } from 'vue';
	import { useRouter } from 'vue-router';
	import { Icon } from '@iconify/vue';
	import { waitForClient } from '../javascript/util';
	import Section from 'picocrank/vue/components/Section.vue';

	const router = useRouter();
	const creating = ref(false);
	const message = ref('');
	const messageType = ref('');
	const form = ref({
		protocol: 'telegram',
		botId: '',
		displayName: '',
		telegramBotToken: '',
		discordToken: '',
		discordAppId: '',
		discordPublicKey: '',
	});

	const isFormValid = computed(() => {
		const f = form.value;
		if (!/^[a-z][a-z0-9_-]{1,30}$/.test(f.botId.trim())) {
			return false;
		}
		if (f.protocol === 'telegram') {
			return f.telegramBotToken.trim().length > 0;
		}
		if (f.protocol === 'discord') {
			return f.discordToken.trim().length > 0;
		}
		return false;
	});

	async function createBot() {
		if (!isFormValid.value || creating.value) {
			return;
		}
		creating.value = true;
		message.value = '';
		try {
			await waitForClient();
			const f = form.value;
			const req = {
				protocol: f.protocol,
				botId: f.botId.trim(),
				displayName: f.displayName.trim() || f.botId.trim(),
			};
			if (f.protocol === 'telegram') {
				req.telegramBotToken = f.telegramBotToken.trim();
			} else {
				req.discordToken = f.discordToken.trim();
				if (f.discordAppId.trim()) {
					req.discordAppId = f.discordAppId.trim();
				}
				if (f.discordPublicKey.trim()) {
					req.discordPublicKey = f.discordPublicKey.trim();
				}
			}
			const res = await window.client.createChatBot(req);
			if (res.standardResponse?.success && res.bot) {
				await router.push({
					name: 'chatBotDetails',
					params: {
						protocol: res.bot.connector || f.protocol,
						botId: res.bot.botId || f.botId.trim(),
					},
				});
				return;
			}
			message.value = res.standardResponse?.message || 'Failed to create chat bot';
			messageType.value = 'error';
		} catch (e) {
			message.value = e.message || 'Failed to create chat bot';
			messageType.value = 'error';
		} finally {
			creating.value = false;
		}
	}
</script>

<style scoped>
	.create-bot-grid {
		display: grid;
		grid-template-columns: minmax(8rem, 12rem) 1fr;
		gap: 0.75rem 1rem;
		align-items: start;
		max-width: 40rem;
	}

	.create-bot-grid label {
		padding-top: 0.35rem;
	}

	.create-bot-grid input,
	.create-bot-grid select {
		width: 100%;
		box-sizing: border-box;
	}

	.field-hint {
		display: block;
		margin-top: 0.25rem;
		color: var(--text-muted, #666);
		font-size: 0.85em;
	}

	.create-bot-actions {
		margin-top: 0.5rem;
	}

	.span-all {
		grid-column: 1 / -1;
	}
</style>
