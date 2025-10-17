<template>
	<Section
		title="Timeline"
		subtitle="This shows the latest posts from your social accounts."
		classes="timeline"
		:padding="false"
	>
		<template #toolbar>
			<button @click="refreshTimeline" :disabled="!clientReady" class="neutral">
				<Icon icon="material-symbols:refresh" />
			</button>
		</template>

		<div v-if="timeline.length === 0">
			<p class="inline-notification note">No posts available.</p>
		</div>
		<table class = "data-table" v-else>
			<thead>
				<tr>
					<th>Social Account</th>
					<th>Campaign</th>
					<th>Content</th>
					<th>Date</th>
					<th>Status</th>
					<th class="medium" style="text-align: right">URL</th>
				</tr>
			</thead>
			<tbody>
				<tr v-if="!clientReady">
					<td colspan="6">Loading timeline...</td>
				</tr>
				<tr v-else v-for="post in pagedTimeline" :key="post.id">
					<td>
						<span class="social-account">
							<Icon :icon="post.socialAccountIcon" />
							{{ post.socialAccountIdentity }}
						</span>
					</td>
					<td>
						<div v-if="post.campaignId != 0">{{ post.campaignName }}</div>
						<div v-else>None</div>
					</td>
					<td>{{ post.content }}</td>
					<td>{{ post.created }}</td>
					<td>
						<span :class="statusClass(post)">{{ statusText(post) }}</span>
					</td>
					<td align="right">
						<a v-if="post.postUrl" :href="post.postUrl" target="_blank">link</a>
						<span v-else>-</span>
					</td>
				</tr>
			</tbody>
		</table>
		<div v-if="timeline.length > 0" style="margin-top: 1rem; display: flex; justify-content: center;">
			<Pagination
				:total="timeline.length"
				:page="currentPage"
				:page-size="pageSize"
				@change="onPageChange"
			/>
		</div>
	</Section>
</template>

<script setup>
	import { ref, onMounted, computed } from 'vue';
	import { waitForClient } from '../javascript/util';
	import { Icon } from '@iconify/vue';
	import Section from 'picocrank/vue/components/Section.vue';
	import Pagination from 'picocrank/vue/components/Pagination.vue';

	const timeline = ref([]);
	const clientReady = ref(false);
	const currentPage = ref(1);
	const pageSize = ref(10);

	const pagedTimeline = computed(() => {
		const start = (currentPage.value - 1) * pageSize.value;
		return timeline.value.slice(start, start + pageSize.value);
	});

	function statusClass(post) {
		// Preferred order: error -> bad, pending/scheduled -> note, completed/success -> good
		if (post.state === 'error') return 'bad';
		if (post.state === 'pending' || post.state === 'scheduled') return 'note';
		if (post.success) return 'good';
		return '';
	}

	function statusText(post) {
		if (post.state === 'error') return 'Error';
		if (post.state === 'pending' || post.state === 'scheduled') return 'Scheduled';
		return post.success ? 'Completed' : 'Unknown';
	}

	async function getTimeline() {
		if (!window.client) {
			return [];
		}

		return await window.client.getTimeline()
			.then((ret) => {
				return ret.posts || [];
			})
			.catch((error) => {
				console.error('Error fetching timeline:', error);
				return [];
			});
	}

	function refreshTimeline() {
		getTimeline().then((posts) => {
			timeline.value = posts;
			currentPage.value = 1;
		});
	}

	function onPageChange(newPage) {
		currentPage.value = newPage;
	}

	onMounted(async () => {
		await waitForClient();
		clientReady.value = true;

		refreshTimeline();
	});
</script>
