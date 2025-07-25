<template>
	<section class = "canned-posts">
		<div class = "flex-row section-header">
			<div class = "fg1">
				<h2>Canned Posts</h2>

				<p>This page shows a list of social accounts that can be used in the chat.</p>
			</div>
			<div role = "toolbar">
				<button @click = "refreshPosts" :disabled = "!clientReady" class = "neutral">
					<Icon icon="material-symbols:refresh" />
				</button>

				<button class = "good" :disabled = "!clientReady" @click = "createCannedPost">
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
							<th>Content</th>
							<th>Created</th>
							<th class = "small"></th>
						</tr>
					</thead>
					<tbody>
					<tr class = "canned-post" v-for = "p in posts" :key = "p.id">
						<td>
							<textarea :id = "'canned-post-' + p.id"
							    v-model = "p.content"
							    @click = "beginEditing(p)"
							    @keyup.enter = "saveCannedPost(p)"
							    @keyup.esc = "cancelEditing(p)"
								:readonly = "!p.editing">{{ p.content }}</textarea>
						</td>
						<td>
							{{ p.createdAt }}
						</td>

						<td align = "right">
							<button @click = "usePost(p)" class = "good">
								<Icon icon="jam:write-f" />
							</button>

							&nbsp;

							<button @click = "deleteCannedPost(p.id)" class = "bad">
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
	import { ref, onMounted, inject } from 'vue';
	import { waitForClient } from '../javascript/util';

	const posts = ref([])
	const clientReady = ref(false)
	const errorMessage = ref("")
	const changeSection = inject('changeSection');
	let startPost = () => {};

	function usePost(p) {
		changeSection('postBox')
	}

	function beginEditing(post) {
		post.editing = true
	}

	function cancelEditing(post) {
		post.editing = false
	}

	function saveCannedPost(post) {
		if (!window.client) {
			errorMessage.value = "Client is not ready."
			return
		}

		window.client.updateCannedPost({
			"id": post.id,
			"content": post.content
			})
			.then(() => {
				console.log('Updated canned post with ID:', post.id)
				post.editing = false
				refreshPosts()
			})
			.catch((error) => {
				errorMessage.value = "Failed to update canned post: " + error.message
				console.error('Error updating canned post:', error)
			})
	}

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
				ret.posts.forEach(post => {
					post.editing = false
				})

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
			posts.value = fetchedPosts
		})
	}

	function createCannedPost() {
		if (!window.client) {
			errorMessage.value = "Client is not ready."
			return
		}

		window.client.createCannedPost({
			"content": "This is a new canned post."
			})
			.then(() => {
				console.log('Created new canned post')
				refreshPosts()
			})
			.catch((error) => {
				errorMessage.value = "Failed to create canned post: " + error.message
				console.error('Error creating canned post:', error)
			})
	}

	onMounted(async () => {
		startPost = inject('startPost');
		console.log('Start post function injected:', startPost)

		await waitForClient()

		clientReady.value = true

		refreshPosts()
	})
</script>

<style scoped>
	textarea {
		width: 90%;
		overflow: scroll;
	}
	textarea[readonly] {
		background-color: transparent;
		border: 1px solid transparent;
	}
</style>
