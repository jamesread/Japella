<template>
	<div class="llm-tool-runs-panel" :class="{ flush }">
		<div v-if="showRunner && tools.length" class="llm-runner" role="toolbar" aria-label="LLM tools">
			<select v-model="selectedToolId" :disabled="!tools.length" aria-label="LLM tool">
				<option disabled :value="0">Select a tool…</option>
				<option v-for="t in tools" :key="t.id" :value="t.id">{{ t.name }}</option>
			</select>
			<button
				type="button"
				class="neutral"
				:disabled="!selectedToolId || !canRun"
				@click="$emit('run', selectedToolId)"
			>
				Run tool
			</button>
		</div>

		<NotificationBlock v-if="errorMessage" type="error" :message="errorMessage" />
		<NotificationBlock
			v-if="!loading && !sessions.length && showRunner"
			type="note"
			message="No LLM tool runs for this post yet."
		/>

		<Section
			v-for="session in sessions"
			:key="session.toolId"
			:title="session.toolName || 'LLM tool run'"
			:subtitle="sessionSubtitle(session)"
			:classes="flush ? 'llm-tool-run' : 'llm-tool-run small'"
		>
			<template #toolbar>
				<div class="llm-session-toolbar" role="toolbar" :aria-label="`${session.toolName} controls`">
					<template v-if="historyLen(session) > 1 && !session.pending">
						<button
							type="button"
							class="inline-icon neutral"
							aria-label="Previous reply"
							title="Previous reply"
							:disabled="!canGoPrev(session)"
							@click="goPrev(session)"
						>
							<Icon icon="material-symbols:chevron-left" />
						</button>
						<span class="llm-history-pos" aria-live="polite">
							{{ historyPosLabel(session) }}
						</span>
						<button
							type="button"
							class="inline-icon neutral"
							aria-label="Next reply"
							title="Next reply"
							:disabled="!canGoNext(session)"
							@click="goNext(session)"
						>
							<Icon icon="material-symbols:chevron-right" />
						</button>
					</template>
					<button
						v-if="canRerunSession(session)"
						type="button"
						class="inline-icon neutral"
						aria-label="Re-run tool"
						title="Re-run with current post text"
						@click="$emit('rerun', session.toolId)"
					>
						<Icon icon="material-symbols:refresh" />
					</button>
				</div>
			</template>

			<div v-if="session.pending" class="llm-run-pending">
				<Icon icon="eos-icons:loading" class="llm-run-spinner" />
				<span>Waiting for response…</span>
				<span class="llm-run-timer" aria-live="polite">{{ session.elapsedLabel || '0s' }}</span>
			</div>
			<template v-else-if="currentReply(session)">
				<pre
					v-if="currentReply(session).status === 'ok'"
					class="llm-run-output"
				>{{ currentReply(session).outputContent }}</pre>
				<NotificationBlock
					v-else
					type="error"
					:message="currentReply(session).errorMessage || 'Tool run failed.'"
				/>
			</template>
		</Section>
	</div>
</template>

<script setup>
	import { ref, watch, onMounted, reactive } from 'vue';
	import { Icon } from '@iconify/vue';
	import Section from 'picocrank/vue/components/Section.vue';
	import NotificationBlock from 'picocrank/vue/components/NotificationBlock.vue';

	const props = defineProps({
		sessions: { type: Array, default: () => [] },
		tools: { type: Array, default: () => [] },
		canRun: { type: Boolean, default: false },
		loading: { type: Boolean, default: false },
		errorMessage: { type: String, default: '' },
		showRunner: { type: Boolean, default: false },
		flush: { type: Boolean, default: false },
		canRerun: { type: Boolean, default: false },
	});

	defineEmits(['run', 'rerun']);

	const selectedToolId = ref(0);
	/** @type {Record<number, number>} history index per tool (0 = newest) */
	const historyIndexByTool = reactive({});

	watch(
		() => props.tools,
		(list) => {
			if (!selectedToolId.value && list?.length) {
				selectedToolId.value = list[0].id;
			}
		},
		{ immediate: true },
	);

	watch(
		() => props.sessions,
		(list) => {
			for (const session of list || []) {
				const len = historyLen(session);
				const cur = historyIndexByTool[session.toolId] ?? 0;
				if (cur >= len) {
					historyIndexByTool[session.toolId] = Math.max(0, len - 1);
				}
				if (session.pending) {
					historyIndexByTool[session.toolId] = 0;
				}
			}
		},
		{ deep: true, immediate: true },
	);

	onMounted(() => {
		if (props.tools?.length && !selectedToolId.value) {
			selectedToolId.value = props.tools[0].id;
		}
	});

	function historyLen(session) {
		return Array.isArray(session?.history) ? session.history.length : 0;
	}

	function historyIndex(session) {
		return historyIndexByTool[session.toolId] ?? 0;
	}

	function currentReply(session) {
		const history = session?.history || [];
		if (!history.length) {
			return null;
		}
		return history[historyIndex(session)] || history[0];
	}

	function canRerunSession(session) {
		return props.canRerun && !session.pending && Number(session.toolId) > 0;
	}

	function canGoPrev(session) {
		return historyIndex(session) < historyLen(session) - 1;
	}

	function canGoNext(session) {
		return historyIndex(session) > 0;
	}

	function goPrev(session) {
		if (!canGoPrev(session)) {
			return;
		}
		historyIndexByTool[session.toolId] = historyIndex(session) + 1;
	}

	function goNext(session) {
		if (!canGoNext(session)) {
			return;
		}
		historyIndexByTool[session.toolId] = historyIndex(session) - 1;
	}

	function historyPosLabel(session) {
		const len = historyLen(session);
		if (len < 1) {
			return '';
		}
		const newestFirstPos = historyIndex(session) + 1;
		return `${newestFirstPos} / ${len}`;
	}

	function formatWhen(value) {
		if (!value) return '';
		try {
			return new Date(value).toLocaleString();
		} catch {
			return value;
		}
	}

	function sessionSubtitle(session) {
		if (session.pending) {
			return `Running · ${session.elapsedLabel || '0s'}`;
		}
		const reply = currentReply(session);
		if (!reply) {
			return '';
		}
		const parts = [];
		const when = formatWhen(reply.created);
		if (when) {
			parts.push(when);
		}
		if (reply.model) {
			parts.push(reply.model);
		}
		if (reply.connectionName) {
			parts.push(`via ${reply.connectionName}`);
		}
		if (historyLen(session) > 1) {
			parts.push(historyPosLabel(session));
		}
		parts.push(reply.status === 'ok' ? 'ok' : 'error');
		return parts.join(' · ');
	}
</script>

<style scoped>
	.llm-tool-runs-panel {
		display: flex;
		flex-direction: column;
		gap: 0.75rem;
		width: 100%;
	}

	.llm-tool-runs-panel.flush :deep(section.llm-tool-run) {
		max-width: none;
		margin-left: 0;
		margin-right: 0;
		width: 100%;
	}

	.llm-session-toolbar {
		display: inline-flex;
		align-items: center;
		gap: 0.2rem;
	}

	.llm-history-pos {
		font-variant-numeric: tabular-nums;
		font-size: 0.85em;
		opacity: 0.8;
		min-width: 2.75rem;
		text-align: center;
	}

	.llm-runner {
		display: flex;
		flex-wrap: wrap;
		gap: 0.5rem;
		align-items: center;
	}

	.llm-runner select {
		min-width: 12rem;
	}

	.llm-run-pending {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		gap: 0.5rem 0.75rem;
	}

	.llm-run-spinner {
		font-size: 1.25rem;
	}

	.llm-run-timer {
		font-variant-numeric: tabular-nums;
		opacity: 0.8;
	}

	.llm-run-output {
		margin: 0;
		white-space: pre-wrap;
		word-break: break-word;
		font-family: inherit;
		font-size: 0.95em;
	}
</style>
