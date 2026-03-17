<template>
	<Section
		title="Canned Posts"
		subtitle="Manage your canned posts here. You can create, edit, and delete canned posts."
		classes="canned-posts"
	>
		<template #toolbar>
			<button @click="refreshPosts" :disabled="!clientReady" class="neutral">
				<Icon icon="material-symbols:refresh" />
			</button>
			<button class="good" :disabled="!clientReady" @click="createCannedPost">
				<Icon icon="material-symbols:add-rounded" />
			</button>
		</template>

		<p class="small">Canned posts are pre-defined posts that can be used in the post box. They are saved in the database and can be used multiple times.</p>

	</Section>

		<div v-if="errorMessage">
			<p class="inline-notification error">{{ errorMessage }}</p>
		</div>
		<div v-else>
			<div v-if="posts.length === 0">
				<p class="inline-notification note">No canned posts available. Please create a new canned post.</p>
			</div>
			<div v-else class="canned-posts-list">
				<section v-for="p in posts" :key="p.id" class="canned-post-item small">
					<PostPreview v-if="!p.editing" :post="formatPostForPreview(p)" class="canned-post-preview" />
					<textarea
						v-else
						:id="'canned-post-' + p.id"
						v-model="p.content"
						@keyup.enter="saveCannedPost(p)"
						@keyup.esc="cancelEditing(p)"
						class="edit-textarea"
						rows="6"
					></textarea>
					<div class="post-meta">
						<span class="post-date">{{ formatFuzzyDate(p.createdAt) }}</span>
						<div class="post-actions">
							<button v-if="!p.editing" @click="usePost(p)" class="good" title="Use this post">
								<Icon icon="jam:write-f" />
								Use
							</button>
							<button v-if="!p.editing" @click="beginEditing(p)" class="neutral" title="Edit post">
								<Icon icon="material-symbols:edit" />
								Edit
							</button>
							<button v-if="p.editing" @click="saveCannedPost(p)" class="good" title="Save changes">
								<Icon icon="material-symbols:save" />
								Save
							</button>
							<button v-if="p.editing" @click="cancelEditing(p)" class="neutral" title="Cancel editing">
								<Icon icon="material-symbols:cancel" />
								Cancel
							</button>
							<button @click="deleteCannedPost(p.id)" class="bad" title="Delete post">
								<Icon icon="material-symbols:delete" />
								Delete
							</button>
						</div>
					</div>
				</section>
			</div>
		</div>
</template>

<script setup>
	import { Icon } from '@iconify/vue';
	import { ref, onMounted, inject } from 'vue';
	import { waitForClient } from '../javascript/util';
	import Section from 'picocrank/vue/components/Section.vue';
	import PostPreview from './PostPreview.vue';

	const posts = ref([])
	const clientReady = ref(false)
	const errorMessage = ref("")

	function usePost(p) {
		// Navigate to post box using router with query parameter
		window.router.push({
			path: '/post',
			query: { cannedPostId: p.id }
		})
	}

	function beginEditing(post) {
		post.originalContent = post.content // Save original content
		post.editing = true
	}

	function cancelEditing(post) {
		post.content = post.originalContent // Restore original content
		post.editing = false
	}

	function formatPostForPreview(post) {
		// Format canned post to match PostPreview expected structure
		return {
			id: post.id,
			content: post.content,
			// No social account info for canned posts
			socialAccountId: null,
			socialAccountIcon: null,
			socialAccountIdentity: null,
		}
	}

	function formatFuzzyDate(dateString) {
		if (!dateString) {
			return '';
		}

		try {
			// Parse the date string (format: "2006-01-02 15:04:05" or similar)
			let date = new Date(dateString.replace(' ', 'T'));
			if (isNaN(date.getTime())) {
				date = new Date(dateString);
			}
			if (isNaN(date.getTime())) {
				return dateString; // Return original if can't parse
			}

			const now = new Date();
			const today = new Date(now.getFullYear(), now.getMonth(), now.getDate());
			const targetDate = new Date(date.getFullYear(), date.getMonth(), date.getDate());
			const diffDays = Math.round((targetDate - today) / (1000 * 60 * 60 * 24));

			// Format time (e.g., "4pm", "2:30pm", "11:15am")
			const hours = date.getHours();
			const minutes = date.getMinutes();
			const ampm = hours >= 12 ? 'pm' : 'am';
			const displayHours = hours % 12 || 12;
			const timeStr = minutes > 0
				? `${displayHours}:${minutes.toString().padStart(2, '0')}${ampm}`
				: `${displayHours}${ampm}`;

			if (diffDays === 0) {
				return `Today at ${timeStr}`;
			} else if (diffDays === 1) {
				return `Tomorrow at ${timeStr}`;
			} else if (diffDays === -1) {
				return `Yesterday at ${timeStr}`;
			} else if (diffDays > 1 && diffDays <= 7) {
				// Within next week
				const dayNames = ['Sunday', 'Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday'];
				return `${dayNames[date.getDay()]} at ${timeStr}`;
			} else if (diffDays < -1 && diffDays >= -7) {
				// Within past week
				const dayNames = ['Sunday', 'Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday'];
				return `Last ${dayNames[date.getDay()]} at ${timeStr}`;
			} else {
				// Further away - show date
				const monthNames = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec'];
				const month = monthNames[date.getMonth()];
				const day = date.getDate();

				// If same year, don't show year
				if (date.getFullYear() === now.getFullYear()) {
					return `${month} ${day} at ${timeStr}`;
				} else {
					return `${month} ${day}, ${date.getFullYear()} at ${timeStr}`;
				}
			}
		} catch (error) {
			console.error('Error formatting fuzzy date:', error);
			return dateString;
		}
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
		console.log('Fetching canned posts...')
		return await window.client.getCannedPosts()
			.then((ret) => {
				console.log('Received canned posts response:', ret)
				ret.posts.forEach(post => {
					post.editing = false
					post.originalContent = post.content // Save original content for cancel
				})

				return ret.posts
			})
			.catch((error) => {
				errorMessage.value = "Failed to fetch canned posts: " + error.message
				console.error('Error fetching canned posts:', error)
				return []
			})
	}

	function refreshPosts() {
		console.log('Refreshing posts...')
		getCannedPosts().then((fetchedPosts) => {
			console.log('Setting posts to:', fetchedPosts)
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
		await waitForClient()

		clientReady.value = true

		refreshPosts()
	})
</script>

<style scoped>
	.canned-posts-list {
		display: flex;
		flex-direction: column;
		gap: 1.5rem;
	}

	.canned-post-preview {
		margin-bottom: 1rem;
	}

	.canned-post-preview :deep(.post-preview) {
		padding: 0;
		margin: 0;
		max-width: none;
	}

	.edit-textarea {
		width: 100%;
		padding: 0.75rem;
		border: 1px solid var(--border-color, #ddd);
		border-radius: 0.25rem;
		font-size: 1rem;
		font-family: inherit;
		resize: vertical;
		min-height: 120px;
	}

	.post-meta {
		display: flex;
		justify-content: space-between;
		align-items: center;
		padding-top: 1rem;
		border-top: 1px solid var(--border-color, #e0e0e0);
	}

	.post-date {
		color: var(--text-secondary, #666);
		font-size: 0.9rem;
	}

	.post-actions {
		display: flex;
		gap: 0.5rem;
		align-items: center;
	}

	@media (prefers-color-scheme: dark) {
		.edit-textarea {
			background-color: #1a1a1a;
			border-color: #505050;
			color: #e0e0e0;
		}

		.post-meta {
			border-top-color: #404040;
		}

		.post-date {
			color: #aaa;
		}
	}
</style>
