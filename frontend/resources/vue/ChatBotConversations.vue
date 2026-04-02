<template>
	<div class="conversations-section">
		<h3>Conversations</h3>
		<p class="conversations-intro">
			<template v-if="isMultiBot">
				Select a thread to read messages and reply. Threads from every configured chat bot are listed together, newest activity first.
			</template>
			<template v-else>
				People who message this bot appear here. Select one to view the thread and reply as the bot.
			</template>
		</p>

		<div v-if="conversationsLoading" class="icon-and-text" style="margin-top: 1em;">
			<Icon icon="eos-icons:loading" width="20" height="20" />
			<span style="margin-left: .5em;">Loading conversations...</span>
		</div>
		<div v-else-if="conversationsError" class="inline-notification error">
			{{ conversationsError }}
		</div>
		<div v-else-if="conversations.length === 0" class="inline-notification note">
			No conversations yet.
		</div>
		<div v-else class="conversation-layout">
			<div class="conversation-list">
				<button
					v-for="conversation in conversations"
					:key="conversation.compositeKey"
					class="conversation-list-item"
					:class="{ active: selectedRow?.compositeKey === conversation.compositeKey }"
					@click="selectConversation(conversation)"
				>
					<div class="conversation-title">{{ conversation.title || 'Unknown sender' }}</div>
					<div v-if="isMultiBot" class="conversation-bot">
						{{ conversation.botName }} · {{ conversation.connector }}
						<span v-if="conversation.identityDisplay"> · {{ conversation.identityDisplay }}</span>
					</div>
					<div class="conversation-preview">{{ conversation.lastMessage || '' }}</div>
				</button>
			</div>
			<div class="conversation-thread">
				<div v-if="messagesLoading" class="icon-and-text">
					<Icon icon="eos-icons:loading" width="20" height="20" />
					<span style="margin-left: .5em;">Loading messages...</span>
				</div>
				<div v-else-if="messagesError" class="inline-notification error">
					{{ messagesError }}
				</div>
				<div v-else>
					<div v-if="messages.length === 0" class="inline-notification note">No messages in this conversation.</div>
					<div v-else ref="messagesListEl" class="messages-list">
						<div
							v-for="message in messages"
							:key="message.id"
							class="message-item"
							:class="message.direction === 'outgoing' ? 'message-outgoing' : 'message-incoming'"
						>
							<div class="message-meta">
								<span>{{ message.direction === 'outgoing' ? 'Bot' : (message.author || 'Unknown') }}</span>
								<span>{{ formatTimestamp(message.timestampUnix) }}</span>
							</div>
							<div class="message-content">{{ message.content }}</div>
						</div>
					</div>

					<div class="reply-box">
						<textarea
							v-model="replyContent"
							placeholder="Write a reply as this bot..."
							:disabled="sendingReply || !selectedRow"
						/>
						<div class="reply-actions">
							<button class="good" :disabled="sendingReply || !replyContent.trim() || !selectedRow" @click="sendReply">
								{{ sendingReply ? 'Sending...' : 'Send reply' }}
							</button>
						</div>
						<div v-if="replyMessage" class="inline-notification" :class="replyMessageType" style="margin-top: 0.75em;">
							{{ replyMessage }}
						</div>
					</div>
				</div>
			</div>
		</div>
	</div>
</template>

<script setup>
	import { ref, watch, onUnmounted, nextTick, computed } from 'vue';
	import { Icon } from '@iconify/vue';

	const props = defineProps({
		/** Single-bot mode: one bot from bot details / per-bot conversations route */
		bot: {
			type: Object,
			default: null,
		},
		/** Multi-bot mode: aggregate conversations from all listed bots (e.g. hub page) */
		bots: {
			type: Array,
			default: null,
		},
		routeIdentity: {
			type: String,
			default: '',
		},
	});

	const isMultiBot = computed(() => Array.isArray(props.bots) && props.bots.length > 0);

	const conversations = ref([]);
	const conversationsLoading = ref(false);
	const conversationsError = ref('');
	const selectedRow = ref(null);
	const messages = ref([]);
	const messagesLoading = ref(false);
	const messagesError = ref('');
	const replyContent = ref('');
	const sendingReply = ref(false);
	const replyMessage = ref('');
	const replyMessageType = ref('');
	const messagesListEl = ref(null);

	/** @type {AbortController | null} */
	let messageStreamAbort = null;
	/** @type {ReturnType<typeof setInterval> | null} */
	let conversationsPollTimer = null;

	function routeIdentityValue() {
		return props.routeIdentity === '_' ? '' : (props.routeIdentity || '');
	}

	function makeCompositeKey(connector, identity, conversationKey) {
		return JSON.stringify({
			c: connector,
			i: identity || '',
			k: conversationKey,
		});
	}

	function mapConversationRow(bot, c) {
		const identityVal = isMultiBot.value ? (bot.identity || '') : routeIdentityValue();
		const convKey = String(c.key || '');
		return {
			compositeKey: makeCompositeKey(bot.connector, identityVal, convKey),
			connector: bot.connector,
			identity: identityVal,
			conversationKey: convKey,
			title: String(c.title || ''),
			lastMessage: String(c.lastMessage || ''),
			lastDirection: String(c.lastDirection || ''),
			lastMessageAtUnix: Number(c.lastMessageAtUnix || 0),
			botName: String(bot.name || bot.connector || 'Bot'),
			identityDisplay: identityVal || '',
		};
	}

	function formatTimestamp(ts) {
		if (!ts) return '';
		const d = new Date(Number(ts) * 1000);
		if (Number.isNaN(d.getTime())) return '';
		return d.toLocaleString();
	}

	function mapApiMessage(m) {
		return {
			id: Number(m.id || 0),
			author: String(m.author || ''),
			content: String(m.content || ''),
			direction: String(m.direction || ''),
			channel: String(m.channel || ''),
			messageId: String(m.messageId || ''),
			timestampUnix: Number(m.timestampUnix || 0),
		};
	}

	function maxMessageId() {
		const ids = messages.value.map(x => x.id);
		return ids.length ? Math.max(...ids) : 0;
	}

	function stopMessageStream() {
		if (messageStreamAbort) {
			messageStreamAbort.abort();
			messageStreamAbort = null;
		}
	}

	function startConversationsPoll() {
		if (conversationsPollTimer) {
			return;
		}
		conversationsPollTimer = setInterval(() => {
			if (isMultiBot.value ? props.bots?.length : props.bot) {
				void refreshConversationSidebar();
			}
		}, 3500);
	}

	function stopConversationsPoll() {
		if (conversationsPollTimer) {
			clearInterval(conversationsPollTimer);
			conversationsPollTimer = null;
		}
	}

	function syncSelectionAfterListUpdate() {
		const list = conversations.value;
		const keys = new Set(list.map(c => c.compositeKey));
		const prev = selectedRow.value;
		if (prev && keys.has(prev.compositeKey)) {
			selectedRow.value = list.find(c => c.compositeKey === prev.compositeKey) || null;
			return;
		}
		if (list.length > 0 && (!prev || !keys.has(prev.compositeKey))) {
			selectedRow.value = list[0];
			void fetchConversationMessages();
		} else if (list.length === 0) {
			selectedRow.value = null;
			messages.value = [];
		}
	}

	async function refreshConversationSidebar() {
		const botList = isMultiBot.value ? props.bots : props.bot ? [props.bot] : [];
		if (!botList.length) {
			return;
		}
		try {
			if (isMultiBot.value) {
				const results = await Promise.all(
					botList.map(async (bot) => {
						const res = await window.client.getBotConversations({
							connector: bot.connector,
							identity: bot.identity || '',
							limit: 200,
						});
						return (res.conversations || []).map(c => mapConversationRow(bot, c));
					}),
				);
				const flat = results.flat();
				flat.sort((a, b) => (b.lastMessageAtUnix || 0) - (a.lastMessageAtUnix || 0));
				conversations.value = flat;
			} else {
				const bot = props.bot;
				const res = await window.client.getBotConversations({
					connector: bot.connector,
					identity: routeIdentityValue(),
					limit: 200,
				});
				conversations.value = (res.conversations || []).map(c => mapConversationRow(bot, c));
			}
			syncSelectionAfterListUpdate();
		} catch (e) {
			console.warn('refreshConversationSidebar', e);
		}
	}

	function scrollMessagesToBottom() {
		const el = messagesListEl.value;
		if (el) {
			el.scrollTop = el.scrollHeight;
		}
	}

	async function runMessageStream() {
		stopMessageStream();
		const row = selectedRow.value;
		if (!row || !row.conversationKey || typeof window.client.streamBotConversationUpdates !== 'function') {
			return;
		}
		const streamKey = row.compositeKey;
		messageStreamAbort = new AbortController();
		const { signal } = messageStreamAbort;
		const lastId = maxMessageId();
		try {
			const stream = window.client.streamBotConversationUpdates(
				{
					connector: row.connector,
					identity: row.identity,
					conversationKey: row.conversationKey,
					lastMessageId: lastId,
				},
				{ signal },
			);
			for await (const chunk of stream) {
				if (selectedRow.value?.compositeKey !== streamKey) {
					break;
				}
				const incoming = chunk.newMessages || [];
				if (incoming.length === 0) {
					continue;
				}
				const existing = new Set(messages.value.map(m => m.id));
				let added = false;
				for (const m of incoming) {
					const rowMsg = mapApiMessage(m);
					if (!existing.has(rowMsg.id)) {
						existing.add(rowMsg.id);
						messages.value.push(rowMsg);
						added = true;
					}
				}
				if (added) {
					messages.value.sort((a, b) => a.id - b.id);
					await nextTick();
					scrollMessagesToBottom();
				}
				void refreshConversationSidebar();
			}
		} catch (e) {
			if (e?.name === 'AbortError' || signal.aborted) {
				return;
			}
			console.error('StreamBotConversationUpdates', e);
		}
	}

	async function fetchConversations() {
		const botList = isMultiBot.value ? props.bots : props.bot ? [props.bot] : [];
		if (!botList.length) {
			conversations.value = [];
			selectedRow.value = null;
			return;
		}

		conversationsLoading.value = true;
		conversationsError.value = '';
		try {
			if (isMultiBot.value) {
				const results = await Promise.all(
					botList.map(async (bot) => {
						const res = await window.client.getBotConversations({
							connector: bot.connector,
							identity: bot.identity || '',
							limit: 200,
						});
						return (res.conversations || []).map(c => mapConversationRow(bot, c));
					}),
				);
				const flat = results.flat();
				flat.sort((a, b) => (b.lastMessageAtUnix || 0) - (a.lastMessageAtUnix || 0));
				conversations.value = flat;
			} else {
				const bot = props.bot;
				const res = await window.client.getBotConversations({
					connector: bot.connector,
					identity: routeIdentityValue(),
					limit: 200,
				});
				conversations.value = (res.conversations || []).map(c => mapConversationRow(bot, c));
			}

			syncSelectionAfterListUpdate();
		} catch (e) {
			conversationsError.value = `Failed to load conversations: ${e.message || e}`;
			conversations.value = [];
			selectedRow.value = null;
		} finally {
			conversationsLoading.value = false;
			if (botList.length) {
				startConversationsPoll();
			}
		}
	}

	async function fetchConversationMessages() {
		const row = selectedRow.value;
		if (!row || !row.conversationKey) {
			messages.value = [];
			return;
		}
		messagesLoading.value = true;
		messagesError.value = '';
		try {
			const res = await window.client.getBotConversationMessages({
				connector: row.connector,
				identity: row.identity,
				conversationKey: row.conversationKey,
				limit: 500,
			});
			messages.value = (res.messages || []).map(mapApiMessage);
		} catch (e) {
			messagesError.value = `Failed to load messages: ${e.message || e}`;
			messages.value = [];
		} finally {
			messagesLoading.value = false;
			if (row && selectedRow.value?.compositeKey === row.compositeKey) {
				void runMessageStream();
				void nextTick().then(scrollMessagesToBottom);
			}
		}
	}

	function selectConversation(row) {
		stopMessageStream();
		selectedRow.value = row;
		replyMessage.value = '';
		void fetchConversationMessages();
	}

	async function sendReply() {
		const row = selectedRow.value;
		if (!row || !row.conversationKey || !replyContent.value.trim()) {
			return;
		}
		sendingReply.value = true;
		replyMessage.value = '';
		replyMessageType.value = '';
		try {
			const res = await window.client.sendBotConversationMessage({
				connector: row.connector,
				identity: row.identity,
				conversationKey: row.conversationKey,
				content: replyContent.value.trim(),
			});
			if (res.standardResponse?.success) {
				replyContent.value = '';
				replyMessage.value = 'Reply sent';
				replyMessageType.value = 'note';
				await fetchConversationMessages();
				await fetchConversations();
			} else {
				replyMessage.value = res.standardResponse?.message || 'Failed to send reply';
				replyMessageType.value = 'error';
			}
		} catch (e) {
			replyMessage.value = `Failed to send reply: ${e.message || e}`;
			replyMessageType.value = 'error';
		} finally {
			sendingReply.value = false;
		}
	}

	function resetAndReload() {
		stopMessageStream();
		stopConversationsPoll();
		selectedRow.value = null;
		messages.value = [];
		replyContent.value = '';
		replyMessage.value = '';
		const botList = isMultiBot.value ? props.bots : props.bot ? [props.bot] : [];
		if (botList.length) {
			void fetchConversations();
		} else {
			conversations.value = [];
		}
	}

	watch(
		() => ({
			multi: isMultiBot.value,
			botConnector: props.bot?.connector,
			routeIdentity: props.routeIdentity,
			botsSig: isMultiBot.value
				? (props.bots || []).map(b => `${b.connector}\0${b.identity || ''}`).join('|')
				: '',
		}),
		() => {
			resetAndReload();
		},
		{ deep: true, immediate: true },
	);

	onUnmounted(() => {
		stopMessageStream();
		stopConversationsPoll();
	});
</script>

<style scoped>
	.conversations-section {
		color: var(--text-color, #1a1a1a);
	}

	.conversations-intro {
		margin-top: 0.5em;
		margin-bottom: 1em;
		color: var(--text-muted, #666);
	}

	.conversation-layout {
		display: grid;
		grid-template-columns: 300px 1fr;
		gap: 1rem;
	}

	.conversation-list {
		border: 1px solid var(--conversations-border, #d8d8d8);
		border-radius: 0.25em;
		overflow: hidden;
		max-height: 520px;
		overflow-y: auto;
		background: var(--conversations-list-surface, #fafafa);
	}

	.conversation-list-item {
		width: 100%;
		text-align: left;
		color: inherit;
		background: var(--conversations-item-bg, #fff);
		border: 0;
		border-bottom: 1px solid var(--conversations-border, #e8e8e8);
		padding: 0.75em;
		cursor: pointer;
	}

	.conversation-list-item:hover {
		background: var(--conversations-item-hover, #f0f4f8);
	}

	.conversation-list-item.active {
		background: var(--conversations-item-active, #e3f2fd);
	}

	.conversation-title {
		font-weight: 600;
		margin-bottom: 0.25em;
		color: var(--text-color, #1a1a1a);
	}

	.conversation-bot {
		font-size: 0.78em;
		color: var(--text-muted, #666);
		margin-bottom: 0.25em;
	}

	.conversation-preview {
		font-size: 0.85em;
		color: var(--text-muted, #666);
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	.conversation-thread {
		border: 1px solid var(--conversations-border, #d8d8d8);
		border-radius: 0.25em;
		padding: 0.75em;
		background: var(--conversations-thread-bg, #fff);
	}

	.messages-list {
		display: flex;
		flex-direction: column;
		gap: 0.5em;
		max-height: 420px;
		overflow-y: auto;
	}

	.message-item {
		padding: 0.65em;
		border-radius: 0.25em;
	}

	.message-incoming {
		background: var(--message-incoming-bg, #f3f4f6);
	}

	.message-outgoing {
		background: var(--message-outgoing-bg, #e8f5e9);
	}

	.message-meta {
		display: flex;
		justify-content: space-between;
		font-size: 0.78em;
		color: var(--text-muted, #666);
		margin-bottom: 0.35em;
	}

	.message-content {
		white-space: pre-wrap;
		word-break: break-word;
		color: var(--text-color, #1a1a1a);
	}

	.reply-box {
		margin-top: 0.75em;
		padding-top: 0.75em;
		border-top: 1px solid var(--conversations-border, #e0e0e0);
	}

	.reply-box textarea {
		width: 100%;
		min-height: 85px;
		background: var(--input-bg, #fff);
		border: 1px solid var(--border-color, #ccc);
		border-radius: 0.25em;
		color: var(--text-color, #1a1a1a);
		padding: 0.5em;
		resize: vertical;
		box-sizing: border-box;
	}

	.reply-box textarea:focus {
		outline: none;
		border-color: var(--primary-color, #1976d2);
		box-shadow: 0 0 0 2px rgba(25, 118, 210, 0.2);
	}

	.reply-box textarea:disabled {
		background: var(--input-disabled-bg, #f0f0f0);
		color: var(--text-muted, #888);
	}

	.reply-actions {
		display: flex;
		justify-content: flex-end;
		margin-top: 0.5em;
	}

	@media (prefers-color-scheme: dark) {
		.conversations-section {
			color: var(--text-color, #f0f0f0);
		}

		.conversations-intro {
			color: var(--text-muted, #aaa);
		}

		.conversation-list {
			--conversations-border: #333;
			--conversations-list-surface: #141414;
			--conversations-item-bg: #1f1f1f;
			--conversations-item-hover: #252525;
			--conversations-item-active: #2b2b2b;
		}

		.conversation-title {
			color: var(--text-color, #f0f0f0);
		}

		.conversation-bot {
			color: var(--text-muted, #aaa);
		}

		.conversation-preview {
			color: var(--text-muted, #aaa);
		}

		.conversation-thread {
			--conversations-border: #333;
			--conversations-thread-bg: #1a1a1a;
		}

		.message-incoming {
			--message-incoming-bg: #2b2b2b;
		}

		.message-outgoing {
			--message-outgoing-bg: #183024;
		}

		.message-meta {
			color: var(--text-muted, #aaa);
		}

		.message-content {
			color: var(--text-color, #f0f0f0);
		}

		.reply-box {
			--conversations-border: #333;
		}

		.reply-box textarea {
			--input-bg: #101010;
			--border-color: #444;
			--text-color: #fff;
			--primary-color: #60a5fa;
			--input-disabled-bg: #1a1a1a;
		}

		.reply-box textarea:focus {
			box-shadow: 0 0 0 2px rgba(96, 165, 250, 0.25);
		}
	}

	@media (max-width: 900px) {
		.conversation-layout {
			grid-template-columns: 1fr;
		}
	}
</style>
