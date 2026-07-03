<template>
	<div v-if="open" class="modal-overlay" @click.self="close">
		<div class="modal">
			<h3>Diagnostic Info</h3>
			<p v-if="post.state" class="diagnostic-status">
				<span :class="['annotation', postStatusClass]">{{ postStatusText }}</span>
			</p>
			<dl class="diagnostic-content">
				<dt v-if="post.id">ID</dt>
				<dd v-if="post.id" class="id-value">{{ post.id }}</dd>
				<dt v-if="post.remoteId">Remote ID</dt>
				<dd v-if="post.remoteId" class="id-value remote-id">{{ post.remoteId }}</dd>
			</dl>
			<p v-if="refetchMessage" class="refetch-message" :class="refetchMessageType">
				{{ refetchMessage }}
			</p>
			<div class="dialog-actions">
				<button
					v-if="showViewDetails && post.id"
					type="button"
					class="neutral"
					@click="viewDetails"
				>
					<Icon icon="mdi:eye" />
					View Post Details
				</button>
				<button
					v-if="canRetry"
					type="button"
					class="good"
					:disabled="retrying"
					@click="retryPost"
				>
					<Icon v-if="retrying" icon="eos-icons:loading" />
					<Icon v-else icon="mdi:refresh" />
					{{ retrying ? 'Retrying…' : 'Retry Post' }}
				</button>
				<button
					v-if="canRefetch"
					type="button"
					class="good"
					:disabled="refetching"
					@click="refetchPost"
				>
					<Icon v-if="refetching" icon="eos-icons:loading" />
					<Icon v-else icon="mdi:refresh" />
					{{ refetching ? 'Refetching…' : 'Refetch' }}
				</button>
				<button type="button" class="neutral" @click="close">Close</button>
			</div>
		</div>
	</div>
</template>

<script setup>
	import { ref, computed } from 'vue';
	import { Icon } from '@iconify/vue';

	const props = defineProps({
		open: {
			type: Boolean,
			default: false,
		},
		post: {
			type: Object,
			required: true,
		},
		refetchable: {
			type: Boolean,
			default: true,
		},
		showViewDetails: {
			type: Boolean,
			default: false,
		},
		retryable: {
			type: Boolean,
			default: false,
		},
		retrying: {
			type: Boolean,
			default: false,
		},
	});

	const emit = defineEmits(['close', 'post-refetched', 'view-details', 'retry-post']);

	const refetching = ref(false);
	const refetchMessage = ref('');
	const refetchMessageType = ref('note');

	const canRefetch = computed(() => props.refetchable && Boolean(props.post.id));
	const canRetry = computed(() => props.retryable && props.post.state === 'error');

	const postStatusText = computed(() => {
		if (props.post.state === 'error') return 'Error';
		if (props.post.state === 'pending' || props.post.state === 'scheduled') return 'Scheduled';
		if (props.post.state === 'completed') return 'Completed';
		return 'Unknown';
	});

	const postStatusClass = computed(() => {
		const text = postStatusText.value;
		if (text === 'Error') return 'bad';
		if (text === 'Completed') return 'good';
		if (text === 'Unknown') return 'warn';
		if (text === 'Scheduled') return 'note';
		return '';
	});

	function close() {
		emit('close');
	}

	function viewDetails() {
		emit('view-details');
		close();
	}

	function retryPost() {
		emit('retry-post');
	}

	async function refetchPost() {
		if (!props.post.id || !window.client || refetching.value) {
			return;
		}

		refetching.value = true;
		refetchMessage.value = '';
		refetchMessageType.value = 'note';

		try {
			const response = await window.client.refetchFeedPost({ id: props.post.id });
			if (!response.standardResponse?.success || !response.post) {
				refetchMessageType.value = 'error';
				refetchMessage.value = response.standardResponse?.message || 'Failed to refetch post.';
				return;
			}

			refetchMessageType.value = 'good';
			refetchMessage.value = 'Post refreshed from social account.';
			emit('post-refetched', response.post);
		} catch (error) {
			console.error('Error refetching feed post:', error);
			refetchMessageType.value = 'error';
			refetchMessage.value = 'Failed to refetch post from social account.';
		} finally {
			refetching.value = false;
		}
	}
</script>

<style scoped>
.diagnostic-content {
	margin-bottom: 1rem;
}

.diagnostic-content dt {
	font-weight: 500;
	font-size: 0.875rem;
	color: var(--text-secondary, #666);
	margin-top: 0.75rem;
	margin-bottom: 0.25rem;
}

.diagnostic-content dt:first-child {
	margin-top: 0;
}

.diagnostic-content dd {
	margin: 0 0 0.75rem 0;
	font-size: 0.875rem;
}

.diagnostic-content dd.remote-id {
	font-size: 0.75rem;
}

.id-value {
	font-family: monospace;
	background-color: var(--bg-secondary, #f5f5f5);
	padding: 0.125rem 0.375rem;
	border-radius: 0.25rem;
	word-break: break-all;
	max-width: 400px;
	overflow-wrap: break-word;
	display: inline-block;
}

.refetch-message {
	margin: 0 0 1rem;
	font-size: 0.875rem;
}

.refetch-message.good {
	color: var(--good-color, #2e7d32);
}

.refetch-message.error {
	color: var(--error-color, #c62828);
}

.refetch-message.note {
	color: var(--text-secondary, #666);
}

.diagnostic-status {
	margin: 0 0 1rem;
}

.annotation {
	display: inline-block;
	padding: 0.25rem 0.5rem;
	border-radius: 0.25rem;
	font-size: 0.875rem;
	font-weight: 500;
	text-align: center;
	min-width: 60px;
}

.annotation.warn {
	background-color: #fff3cd;
	color: #856404;
	border: 1px solid #ffeaa7;
}

.annotation.bad {
	background-color: #f8d7da;
	color: #721c24;
	border: 1px solid #f5c6cb;
}

.annotation.good {
	background-color: #d4edda;
	color: #155724;
	border: 1px solid #c3e6cb;
}

.annotation.note {
	background-color: #fff3cd;
	color: #856404;
	border: 1px solid #ffeaa7;
}

.modal-overlay {
	position: fixed;
	top: 0;
	left: 0;
	width: 100%;
	height: 100%;
	background-color: rgba(0, 0, 0, 0.5);
	display: flex;
	justify-content: center;
	align-items: center;
	z-index: 1000;
}

.modal {
	background: white;
	padding: 2rem;
	border-radius: 0.5rem;
	box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1);
	min-width: 400px;
	max-width: 500px;
}

.modal h3 {
	margin: 0 0 1rem 0;
	color: #333;
}

.dialog-actions {
	display: flex;
	gap: 0.5rem;
	justify-content: flex-end;
	margin-top: 1.5rem;
}

.dialog-actions button {
	padding: 0.5rem 1rem;
	border-radius: 0.25rem;
	border: none;
	cursor: pointer;
	font-size: 0.9rem;
	display: flex;
	align-items: center;
	gap: 0.5rem;
	transition: all 0.2s ease;
}

.dialog-actions button:disabled {
	opacity: 0.6;
	cursor: not-allowed;
}

.dialog-actions button.neutral {
	background-color: #f0f0f0;
	color: #333;
}

.dialog-actions button.neutral:hover:not(:disabled) {
	background-color: #e0e0e0;
}

.dialog-actions button.good {
	background-color: #4caf50;
	color: white;
}

.dialog-actions button.good:hover:not(:disabled) {
	background-color: #43a047;
}
</style>
