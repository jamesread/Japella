<template>
	<div v-if="post.canLike || post.canViewComments" class="post-interactions" role="group" aria-label="Post actions">
		<button
			v-if="post.canLike"
			type="button"
			class="neutral small interaction"
			:class="{ liked }"
			:aria-pressed="liked"
			:title="liked ? 'Unlike' : 'Like'"
			:disabled="liking"
			@click="toggleLike"
		>
			<Icon :icon="liked ? 'mdi:heart' : 'mdi:heart-outline'" aria-hidden="true" />
			<span>{{ likeCount ? likeCount.toLocaleString() : 'Like' }}</span>
		</button>
		<button
			v-if="post.canViewComments"
			type="button"
			class="small interaction"
			:class="commented ? 'good' : 'neutral'"
			:title="commented ? 'Show comments (you have commented)' : 'Show comments'"
			:aria-label="`${replyCount} ${replyCount === 1 ? 'comment' : 'comments'}${commented ? ', including yours' : ''}`"
			@click="showThread = true"
		>
			<Icon :icon="commented ? 'mdi:comment-check-outline' : 'mdi:comment-outline'" aria-hidden="true" />
			<span>{{ replyCount.toLocaleString() }}</span>
		</button>
		<span v-if="error" class="interaction-error" role="alert">{{ error }}</span>

		<PostThreadDialog
			v-if="showThread"
			:post-ref="postRef"
			:can-comment="Boolean(post.canComment)"
			@loaded="onThreadLoaded"
			@commented="onCommented"
			@close="showThread = false"
		/>
	</div>
</template>

<script setup>
	import { ref, computed, watch } from 'vue';
	import { Icon } from '@iconify/vue';
	import PostThreadDialog from './PostThreadDialog.vue';

	const props = defineProps({
		post: {
			type: Object,
			required: true,
		},
	});

	const liked = ref(Boolean(props.post.liked));
	const likeCount = ref(props.post.likeCount || 0);
	const liking = ref(false);
	const error = ref('');
	const showThread = ref(false);
	const replyCount = ref(props.post.replyCount || 0);
	const commented = ref(Boolean(props.post.commented));

	watch(() => [props.post.liked, props.post.likeCount], () => {
		liked.value = Boolean(props.post.liked);
		likeCount.value = props.post.likeCount || 0;
	});

	watch(() => [props.post.replyCount, props.post.commented], () => {
		replyCount.value = props.post.replyCount || 0;
		commented.value = Boolean(props.post.commented);
	});

	// The live thread is fresher than the counts stored with the feed post.
	function onThreadLoaded({ replyCount: count, commented: hasCommented }) {
		replyCount.value = count;
		commented.value = commented.value || hasCommented;
	}

	function onCommented() {
		replyCount.value += 1;
		commented.value = true;
	}

	const SNIPPET_CHARS = 300;

	function plainSnippet(html) {
		if (!html) {
			return '';
		}
		const text = (new DOMParser().parseFromString(html, 'text/html').body.textContent || '').trim();
		return text.slice(0, SNIPPET_CHARS);
	}

	const postRef = computed(() => ({
		socialAccountId: props.post.socialAccountId,
		feedPostId: props.post.id || 0,
		remoteId: props.post.remoteId || '',
		remoteUrl: props.post.remoteUrl || '',
		interactionRemoteId: props.post.interactionRemoteId || '',
		// Recorded with the like or comment in the account's recent interactions.
		snippet: plainSnippet(props.post.content),
		authorName: props.post.originalAuthorName || props.post.authorName || '',
	}));

	async function toggleLike() {
		const next = !liked.value;
		const before = { liked: liked.value, likeCount: likeCount.value };
		// Optimistic: update now, put it back if the network refuses.
		liked.value = next;
		likeCount.value = Math.max(0, likeCount.value + (next ? 1 : -1));
		liking.value = true;
		error.value = '';
		try {
			const res = await window.client.setPostLiked({ post: postRef.value, liked: next });
			liked.value = Boolean(res.liked);
			likeCount.value = res.likeCount || 0;
		} catch (e) {
			liked.value = before.liked;
			likeCount.value = before.likeCount;
			error.value = e.rawMessage || e.message || String(e);
		} finally {
			liking.value = false;
		}
	}
</script>

<style scoped>
	.post-interactions {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		gap: 0.4rem;
		margin-top: 0.5rem;
	}

	.interaction {
		display: inline-flex;
		align-items: center;
		gap: 0.3rem;
		padding: 0.2em 0.6em;
		font-variant-numeric: tabular-nums;
	}

	.interaction.liked {
		color: #e0245e;
	}

	.interaction-error {
		font-size: 0.85em;
		color: var(--karma-bad-fg, #b00020);
	}
</style>
