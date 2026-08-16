<template>
	<Section
		title="Canned Posts"
		subtitle="Manage your canned posts here. You can create, edit, and delete canned posts."
		classes="canned-posts"
		:padding="false"
	>
		<template #toolbar>
			<button @click="refreshPosts" :disabled="!clientReady" class="neutral">
				<Icon icon="material-symbols:refresh" />
			</button>
			<button class="good" :disabled="!clientReady" @click="createCannedPost">
				<Icon icon="material-symbols:add-rounded" />
			</button>
		</template>

		<p class="small section-intro">Canned posts are pre-defined posts that can be used in the post box. They are saved in the database and can be used multiple times.</p>

	</Section>

	<div v-if="errorMessage">
		<p class="inline-notification error">{{ errorMessage }}</p>
	</div>
	<div v-else-if="posts.length === 0">
		<p class="inline-notification note">No canned posts available. Please create a new canned post.</p>
	</div>
	<template v-else>
		<section
			v-for="p in posts"
			:key="p.id"
			class="small canned-post-item"
			:data-canned-post-id="p.id"
		>
		<PostPreview v-if="!p.editing" :post="formatPostForPreview(p)" />
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
				<button
					v-if="!p.editing"
					type="button"
					class="inline-icon good"
					title="Use this post"
					@click="usePost(p)"
				>
					<HugeiconsIcon
						:icon="EditIcon"
						width="1em"
						height="1em"
						:strokeWidth="iconStrokeWidth"
						aria-hidden="true"
					/>
					<span>Use</span>
				</button>
				<button
					v-if="!p.editing"
					type="button"
					class="inline-icon neutral"
					title="Edit post"
					@click="beginEditing(p)"
				>
					<HugeiconsIcon
						:icon="EditIcon"
						width="1em"
						height="1em"
						:strokeWidth="iconStrokeWidth"
						aria-hidden="true"
					/>
					<span>Edit</span>
				</button>
				<button
					v-if="p.editing"
					type="button"
					class="inline-icon good"
					title="Save changes"
					@click="saveCannedPost(p)"
				>
					<HugeiconsIcon
						:icon="SaveIcon"
						width="1em"
						height="1em"
						:strokeWidth="iconStrokeWidth"
						aria-hidden="true"
					/>
					<span>Save</span>
				</button>
				<button
					v-if="p.editing"
					type="button"
					class="inline-icon neutral"
					title="Cancel editing"
					@click="cancelEditing(p)"
				>
					<HugeiconsIcon
						:icon="CancelCircleIcon"
						width="1em"
						height="1em"
						:strokeWidth="iconStrokeWidth"
						aria-hidden="true"
					/>
					<span>Cancel</span>
				</button>
				<button
					type="button"
					class="inline-icon bad"
					title="Delete post"
					@click="deleteCannedPost(p.id)"
				>
					<HugeiconsIcon
						:icon="Delete02Icon"
						width="1em"
						height="1em"
						:strokeWidth="iconStrokeWidth"
						aria-hidden="true"
					/>
					<span>Delete</span>
				</button>
			</div>
		</div>
		</section>
	</template>
</template>

<script setup>
	import { Icon } from '@iconify/vue';
	import { HugeiconsIcon } from '@hugeicons/vue';
	import {
		CancelCircleIcon,
		Delete02Icon,
		EditIcon,
		SaveIcon,
	} from '@hugeicons/core-free-icons';
	import { ref, onMounted, inject } from 'vue';
	import { waitForClient } from '../javascript/util';
	import Section from 'picocrank/vue/components/Section.vue';
	import PostPreview from './PostPreview.vue';

	const iconStrokeWidth = 2.5;

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
	.section-intro {
		margin: 0;
		padding: 0 1em 1em;
	}

	section {
		margin-bottom: 1rem;
		padding: 0;
	}

	.canned-post-item :deep(.post-preview) {
		margin-bottom: 0;
	}

	.edit-textarea {
		width: calc(100% - 2rem);
		margin: 1rem;
		box-sizing: border-box;
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
		padding: 0 1rem 1rem;
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

	html[data-theme="dark"] {
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
