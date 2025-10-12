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
		<table v-else>
			<thead>
				<tr>
					<th>Social Account</th>
					<th>Campaign</th>
					<th>Content</th>
					<th>Date</th>
					<th class="medium" style="text-align: right">URL</th>
				</tr>
			</thead>
			<tbody>
				<tr v-if="!clientReady">
					<td colspan="5">Loading timeline...</td>
				</tr>
				<tr v-else v-for="post in timeline" :key="post.id">
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
					<td align="right">
						<a :href="post.postUrl" target="_blank">link</a>
					</td>
				</tr>
			</tbody>
		</table>
	</Section>
</template>

<script setup>
	import { ref, onMounted } from 'vue';
	import { waitForClient } from '../javascript/util';
	import { Icon } from '@iconify/vue';
	import Section from 'picocrank/vue/components/Section.vue';

	const timeline = ref([]);
	const clientReady = ref(false);

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
		});
	}

	onMounted(async () => {
		await waitForClient();
		clientReady.value = true;

		refreshTimeline();
	});
</script>
