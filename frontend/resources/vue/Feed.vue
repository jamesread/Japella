<template>
	<Section
		title="Feed"
		subtitle="Recent posts from your connected social accounts."
		classes="feed"
		:padding="false"
	>
		<template #toolbar>
			<button @click="refreshFeed" :disabled="!clientReady" class="neutral">
				<Icon icon="material-symbols:refresh" />
			</button>
		</template>

		<Loading v-if="feedLoading" message="Loading feed..." :centered="true" />

		<Pagination
			:total="feed.length"
			v-model:page="currentPage"
			v-model:pageSize="pageSize"
		/>
	</Section>

	<section class="small" v-for="post in pagedFeed" :key="post.id">
		<PostBoost v-if="isBoost(post)" :post="post" />
		<PostPreview v-else :post="post" />
	</section>

</template>

<script setup>
	import { ref, onMounted, computed } from 'vue';
	import { waitForClient } from '../javascript/util';
	import { Icon } from '@iconify/vue';
	import Section from 'picocrank/vue/components/Section.vue';
	import Pagination from 'picocrank/vue/components/Pagination.vue';
	import Loading from './Loading.vue';
	import PostPreview from './PostPreview.vue';
	import PostBoost from './PostBoost.vue';

	const feed = ref([]);
	const clientReady = ref(false);
	const currentPage = ref(1);
	const pageSize = ref(10);
	const feedLoading = ref(true);

	const pagedFeed = computed(() => {
		const start = (currentPage.value - 1) * pageSize.value;
		return feed.value.slice(start, start + pageSize.value);
	});

	async function getFeed() {
		if (!window.client) {
			return [];
		}

		try {
			const response = await window.client.getFeed();
			return response.posts || [];
		} catch (error) {
			console.error('Error fetching feed:', error);
			return [];
		}
	}

	function refreshFeed() {
		feedLoading.value = true;
		getFeed().then((posts) => {
			feed.value = posts;
			currentPage.value = 1;
		}).catch(error => {
			console.error("Error fetching feed:", error);
			feed.value = [];
		}).finally(() => {
			feedLoading.value = false;
		});
	}

	function isBoost(post) {
		// Check if remoteId contains ActivityStreams JSON
		if (post.remoteId) {
			try {
				const parsed = JSON.parse(post.remoteId);
				if (parsed && parsed['@context'] && parsed.type === 'Announce') {
					return true;
				}
			} catch (e) {
				// Not JSON, continue checking other indicators
			}
		}

		// Check if remoteUrl indicates a boost activity (contains "/activity" or "/statuses/.../activity")
		if (post.remoteUrl && (
			post.remoteUrl.includes('/activity') ||
			post.remoteUrl.match(/\/statuses\/\d+\/activity/)
		)) {
			// If content is empty or very short, it's likely a boost
			// Boosts typically have empty content or just whitespace
			const content = (post.content || '').trim();
			if (content === '' || content.length < 50) {
				return true;
			}
		}

		// Check if content is empty but we have a remoteUrl (typical boost pattern)
		const content = (post.content || '').trim();
		if (content === '' && post.remoteUrl) {
			// Check if remoteUrl looks like a Mastodon status URL
			if (post.remoteUrl.includes('mastodon') || 
			    post.remoteUrl.includes('fosstodon') ||
			    post.remoteUrl.match(/https?:\/\/[^\/]+\/(users|@)[^\/]+\/statuses\/\d+/)) {
				return true;
			}
		}

		return false;
	}

	onMounted(async () => {
		await waitForClient();
		clientReady.value = true;
		refreshFeed();
	});
</script>

<style scoped>
	.feed-container {
		padding: 1rem;
	}

	.feed-post {
		margin-bottom: 1rem;
	}

	section {
		margin-bottom: 1rem;
		padding: 0;
	}
</style>
