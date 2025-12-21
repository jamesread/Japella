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
			:page="currentPage"
			:page-size="pageSize"
			@change="onPageChange"
		/>
	</Section>

	<section class="small" v-for="post in pagedFeed" :key="post.id">
		<PostPreview :post="post" />
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

	function onPageChange(newPage) {
		currentPage.value = newPage;
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
	}
</style>
