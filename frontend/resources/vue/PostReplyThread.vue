<template>
	<ol class="reply-thread" :class="{ nested: depth > 0, flat: depth >= maxIndentDepth }">
		<li v-for="reply in replies" :key="reply.remoteId" class="reply">
			<article class="reply-body" :aria-label="`Reply from ${displayName(reply)}`">
				<div class="reply-header">
					<FeedAuthorAvatar :avatar-url="reply.authorAvatarUrl" :alt-text="displayName(reply)" />
					<span class="reply-author">{{ displayName(reply) }}</span>
					<span v-if="reply.authorHandle && reply.authorName" class="reply-handle">{{ reply.authorHandle }}</span>
					<time v-if="reply.postedAt" class="reply-time" :datetime="reply.postedAt">
						{{ formatHumanDateTime(reply.postedAt) }}
					</time>
					<a
						v-if="reply.remoteUrl"
						:href="reply.remoteUrl"
						target="_blank"
						rel="noopener noreferrer"
						class="reply-link"
						title="Open reply"
					>
						<Icon icon="material-symbols:open-in-new" aria-hidden="true" />
						<span class="visually-hidden">Open reply</span>
					</a>
				</div>
				<p class="reply-content">{{ reply.content }}</p>
			</article>

			<PostReplyThread
				v-if="reply.replies && reply.replies.length"
				:replies="reply.replies"
				:depth="depth + 1"
			/>
		</li>
	</ol>
</template>

<script setup>
	import { Icon } from '@iconify/vue';
	import FeedAuthorAvatar from './FeedAuthorAvatar.vue';
	import { formatHumanDateTime } from '../javascript/util';

	defineOptions({ name: 'PostReplyThread' });

	defineProps({
		replies: {
			type: Array,
			default: () => [],
		},
		depth: {
			type: Number,
			default: 0,
		},
	});

	// Beyond this depth replies stop indenting so long chains stay readable.
	const maxIndentDepth = 5;

	function displayName(reply) {
		return reply.authorName || reply.authorHandle || 'Unknown';
	}
</script>

<style scoped>
	.reply-thread {
		list-style: none;
		margin: 0;
		padding: 0;
	}

	.reply-thread.nested {
		margin-left: 1.1rem;
		padding-left: 0.9rem;
		border-left: 2px solid var(--border-color, rgba(127, 127, 127, 0.25));
	}

	.reply-thread.nested.flat {
		margin-left: 0;
		padding-left: 0;
		border-left: none;
	}

	.reply {
		margin: 0;
	}

	.reply-body {
		padding: 0.6rem 0;
	}

	.reply-header {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		gap: 0.25rem 0.5rem;
		font-size: 0.9rem;
	}

	.reply-author {
		font-weight: 600;
	}

	.reply-handle,
	.reply-time {
		color: var(--text-secondary, #666);
	}

	.reply-link {
		display: inline-flex;
		color: var(--text-secondary, #666);
	}

	.reply-content {
		margin: 0.35rem 0 0;
		white-space: pre-wrap;
		overflow-wrap: anywhere;
	}

	.visually-hidden {
		position: absolute;
		width: 1px;
		height: 1px;
		overflow: hidden;
		clip: rect(0 0 0 0);
		white-space: nowrap;
	}
</style>
