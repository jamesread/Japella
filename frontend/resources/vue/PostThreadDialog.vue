<template>
	<dialog ref="dialog" class="dialog post-thread-dialog" aria-labelledby="post-thread-title" @close="emit('close')">
		<h2 id="post-thread-title">Comments</h2>

		<Loading v-if="loading" message="Loading comments…" />
		<NotificationBlock v-else-if="error" type="error" :message="error" />
		<p v-else-if="!replies.length" class="subtle">No comments yet.</p>
		<div v-else class="post-thread-body">
			<p class="subtle">{{ replyCount.toLocaleString() }} {{ replyCount === 1 ? 'comment' : 'comments' }}</p>
			<PostReplyThread :replies="replies" />
		</div>

		<form v-if="canComment" class="post-thread-compose" @submit.prevent="submitComment">
			<label for="post-thread-comment">Add a comment</label>
			<textarea
				id="post-thread-comment"
				v-model="draft"
				rows="3"
				placeholder="Write a comment…"
				:disabled="sending"
				@keydown.ctrl.enter.prevent="submitComment"
				@keydown.meta.enter.prevent="submitComment"
			/>
			<NotificationBlock v-if="sendError" type="error" :message="sendError" />
			<div class="post-thread-compose-actions">
				<span class="subtle">Posted publicly as the account this post came through.</span>
				<button type="submit" class="good" :disabled="sending || !draft.trim()">
					{{ sending ? 'Posting…' : 'Comment' }}
				</button>
			</div>
		</form>

		<div class="post-thread-footer">
			<button type="button" class="neutral" @click="dialog?.close()">Close</button>
		</div>
	</dialog>
</template>

<script setup>
	import { ref, onMounted } from 'vue';
	import NotificationBlock from 'picocrank/vue/components/NotificationBlock.vue';
	import Loading from './Loading.vue';
	import PostReplyThread from './PostReplyThread.vue';

	const props = defineProps({
		// { socialAccountId, feedPostId, remoteId, remoteUrl, interactionRemoteId }
		postRef: {
			type: Object,
			required: true,
		},
		canComment: {
			type: Boolean,
			default: false,
		},
	});

	const emit = defineEmits(['close', 'loaded', 'commented']);

	const draft = ref('');
	const sending = ref(false);
	const sendError = ref('');

	async function submitComment() {
		if (sending.value || !draft.value.trim()) {
			return;
		}
		sending.value = true;
		sendError.value = '';
		try {
			const res = await window.client.postComment({ post: props.postRef, content: draft.value });
			if (res.reply) {
				replies.value = [...replies.value, res.reply];
			}
			replyCount.value += 1;
			error.value = '';
			draft.value = '';
			emit('commented');
		} catch (e) {
			sendError.value = e.rawMessage || e.message || String(e);
		} finally {
			sending.value = false;
		}
	}

	const dialog = ref(null);
	const loading = ref(true);
	const error = ref('');
	const replies = ref([]);
	const replyCount = ref(0);

	onMounted(async () => {
		dialog.value?.showModal();
		try {
			const res = await window.client.getPostThread({ post: props.postRef });
			replies.value = res.replies || [];
			replyCount.value = res.replyCount || 0;
			emit('loaded', { replyCount: replyCount.value, commented: Boolean(res.commented) });
		} catch (e) {
			error.value = e.rawMessage || e.message || String(e);
		} finally {
			loading.value = false;
		}
	});
</script>

<style scoped>
	.post-thread-dialog {
		width: min(40rem, calc(100vw - 2rem));
	}

	.post-thread-body {
		max-height: 65vh;
		overflow-y: auto;
	}

	.post-thread-body > p {
		margin: 0 0 0.5rem;
	}

	.post-thread-compose {
		display: flex;
		flex-direction: column;
		gap: 0.4rem;
		margin-top: 1rem;
		padding-top: 1rem;
		border-top: 1px solid var(--border-color, rgba(127, 127, 127, 0.25));
	}

	.post-thread-compose textarea {
		width: 100%;
		min-height: 4.5em;
		resize: vertical;
	}

	.post-thread-compose-actions {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 0.75rem;
		flex-wrap: wrap;
	}

	.post-thread-compose-actions .subtle {
		font-size: 0.85em;
	}

	.post-thread-footer {
		display: flex;
		justify-content: flex-end;
		margin-top: 1rem;
	}
</style>
