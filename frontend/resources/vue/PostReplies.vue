<template>
	<Section
		title="Replies"
		:subtitle="subtitle"
		:icon="BubbleChatIcon"
		classes="post-replies"
		id="replies"
	>
		<Loading v-if="loading" message="Loading replies..." />
		<NotificationBlock v-else-if="error" type="error" :message="error" />
		<NotificationBlock
			v-else-if="!report.supported"
			type="note"
			message="Replies are not available for this social network."
		/>
		<NotificationBlock
			v-else-if="!report.trackingEnabled && !report.replies.length"
			type="note"
			message="Reply fetching is off. Set post_engagement_max_age_days in Settings to turn it on."
		/>
		<p v-else-if="!report.replies.length" class="no-replies">
			{{ report.lastFetchedAt ? 'No replies yet.' : 'Replies have not been fetched yet; they are checked along with engagement.' }}
		</p>
		<PostReplyThread v-else :replies="report.replies" />
	</Section>
</template>

<script setup>
	import { ref, computed, watch, nextTick } from 'vue';
	import { useRoute } from 'vue-router';
	import { BubbleChatIcon } from '@hugeicons/core-free-icons';
	import Section from 'picocrank/vue/components/Section.vue';
	import NotificationBlock from 'picocrank/vue/components/NotificationBlock.vue';
	import Loading from './Loading.vue';
	import PostReplyThread from './PostReplyThread.vue';
	import { waitForClient, formatHumanDateTime } from '../javascript/util';

	const props = defineProps({
		postId: {
			type: Number,
			required: true,
		},
	});

	const route = useRoute();

	const loading = ref(true);
	const error = ref('');
	const report = ref({ supported: false, trackingEnabled: false, replies: [], replyCount: 0, lastFetchedAt: '' });

	const subtitle = computed(() => {
		const r = report.value;
		if (!r.replyCount) {
			return 'The conversation under this post.';
		}
		const count = `${r.replyCount} ${r.replyCount === 1 ? 'reply' : 'replies'}`;
		return r.lastFetchedAt ? `${count} · updated ${formatHumanDateTime(r.lastFetchedAt)}` : count;
	});

	async function load() {
		loading.value = true;
		error.value = '';
		try {
			await waitForClient();
			report.value = await window.client.getPostReplies({ postId: props.postId });
		} catch (e) {
			error.value = `Failed to load replies: ${e.message || e}`;
		} finally {
			loading.value = false;
		}

		// Links from recent interactions land on #replies; the section only
		// exists once the post has loaded, so scroll here rather than in the router.
		if (route.hash === '#replies') {
			await nextTick();
			document.getElementById('replies')?.scrollIntoView({ behavior: 'smooth', block: 'start' });
		}
	}

	watch(() => props.postId, load, { immediate: true });
</script>

<style scoped>
	.no-replies {
		margin: 0;
		color: var(--text-secondary, #666);
	}
</style>
