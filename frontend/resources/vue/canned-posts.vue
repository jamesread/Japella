<template>
	<section class = "canned-posts">
		<div class = "flex-row">
			<div class = "fg1">
				<h2>Canned Posts</h2>

				<p>This page shows a list of social accounts that can be used in the chat.</p>
			</div>
			<div role = "toolbar">
				<button @click = "refreshPosts" :disabled = "!clientReady">
					<Icon icon="material-symbols:refresh" />
				</button>

				<button>
					<Icon icon="material-symbols:add-rounded" />
				</button>
			</div>
		</div>

		<div v-if = "errorMessage">
			<p class = "inline-notification error">{{ errorMessage }}</p>
		</div>
		<div v-else>
			<div v-if = "posts.length === 0" class = "empty">
				<p class = "inline-notification note">No posts available.</p>
			</div>
			<div v-else class = "empty">
				<table>
					<thead>
						<tr>
							<th>ID</th>
							<th>Content</th>
							<th>Created</th>
							<th class = "small">Actions</th>
						</tr>
					</thead>
					<tbody>
					<tr class = "canned-post" v-for = "p in posts" :key = "p.id">
						<td class = "small">
							{{ p.id }}
						</td>
						<td>
							<textarea :id = "'canned-post-' + p.id" readonly>{{ p.content }}</textarea>
						</td>
						<td>
							{{ p.createdAt }}
						</td>

						<td>
							<button @click = "deleteCannedPost(p.id)">
								<Icon icon="material-symbols:delete" />
							</button>
						</td>
					</tr>
					</tbody>
				</table>
			</div>
		</div>
	</section>
</template>

<script setup>
	import { Icon } from '@iconify/vue';
	import { ref, onMounted } from 'vue';
	import { waitForClient } from '../javascript/util';

	const posts = ref([])
	const clientReady = ref(false)
	const errorMessage = ref("")

	function deleteCannedPost(id) {
		if (!window.client) {
			errorMessage.value = "Client is not ready."
			return
		}

		window.client.deleteCannedPost({
			"id": id
			})
			.then(() => {
				console.log('Deleted canned post with ID:', id)
				refreshPosts()
			})
			.catch((error) => {
				errorMessage.value = "Failed to delete canned post: " + error.message
				console.error('Error deleting canned post:', error)
			})
	}

	async function getCannedPosts() {
		return await window.client.getCannedPosts()
			.then((ret) => {
				return ret.posts
			})
			.catch((error) => {
				errorMessage.value = "Failed to fetch social accounts: " + error.message
				console.error('Error fetching social accounts:', error)
				return []
			})
	}

	function refreshPosts() {
		getCannedPosts().then((fetchedPosts) => {
			console.log('Refreshing posts:', fetchedPosts)
			posts.value = fetchedPosts
		})
	}

	onMounted(async () => {
		await waitForClient()

		clientReady.value = true

		refreshPosts()
	})
</script>
