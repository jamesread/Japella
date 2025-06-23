<template>
	<section class="timeline">
		<div class = "flex-row">
			<div class="fg1">
				<h2>Timeline</h2>
				<p>This shows the latest posts from your social accounts.</p>
			</div>

			<div role="toolbar">
				<button @click="refreshTimeline" :disabled="!clientReady" class = "neutral">
					<Icon icon="material-symbols:refresh" />
				</button>
			</div>
		</div>

		<table v-if="timeline.length" class = "data-table">
			<thead>
				<tr>
					<th>Social Account</th>
					<th>Content</th>
					<th>Date</th>
					<th>URL</th>
				</tr>
			</thead>
			<tbody>
				<tr v-if="!clientReady">
					<td colspan="3">Loading timeline...</td>
				</tr>
				<tr v-else v-for="post in timeline" :key="post.id">
					<td>
						<span class="social-account">
							<Icon :icon="post.socialAccountIcon" />
							{{ post.socialAccountIdentity }}
						</span>
					</td>
					<td>{{ post.content }}</td>
					<td>{{ post.created }}</td>
					<td><a :href = "post.postUrl">link</a></td>
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
	import { Icon } from '@iconify/vue';

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
