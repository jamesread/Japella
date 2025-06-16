<template>
	<section class="timeline">
		<h2>Timeline</h2>
		<p>This shows the latest posts from your social accounts.</p>

		<table v-if="timeline.length">
			<thead>
				<tr>
					<th>Title</th>
					<th>Content</th>
					<th>Date</th>
				</tr>
			</thead>
			<tbody>
				<tr v-if="!clientReady">
					<td colspan="3">Loading timeline...</td>
				</tr>
				<tr v-else v-for="post in timeline" :key="post.id">
					<td>{{ post.title }}</td>
					<td>{{ post.content }}</td>
					<td>{{ post.date }}</td>
				</tr>
			</tbody>
		</table>

		<div v-if="!timeline.length" class="no-posts">
			<InlineNotification type="note" message="No posts available." />
		</div>
	</section>

</template>

<script setup>
	import { ref, onMounted } from 'vue';
	import { waitForClient } from '../javascript/util';
	import InlineNotification from './inline-notification.vue';

	const timeline = ref([]);
	const clientReady = ref(false);

	async function getTimeline() {
		if (!window.client) {
			console.error("Client is not ready.")
			return [];
		}

		return await window.client.getTimeline()
			.then((ret) => {
				return ret.timeline;
			})
			.catch((error) => {
				console.error('Error fetching timeline:', error);
				return [];
			});
	}

	onMounted(async () => {
		await waitForClient();
		clientReady.value = true;

		timeline.value = await getTimeline();
	});
</script>
