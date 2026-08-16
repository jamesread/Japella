<template>
	<Teleport to="body">
	<div v-if="open && item" class="modal-overlay" @click.self="close">
		<div class="media-preview-modal" role="dialog" aria-modal="true" :aria-label="item.filename">
			<button type="button" class="close-button neutral" aria-label="Close" @click="close">×</button>

			<div class="media-preview-layout">
				<div class="media-preview-main">
					<img
						v-if="isImage(item.filename)"
						:src="item.url"
						:alt="item.filename"
						class="media-preview-image"
					/>
					<video
						v-else-if="isVideo(item.filename)"
						:src="item.url"
						class="media-preview-video"
						controls
						playsinline
					/>
					<div v-else class="media-preview-fallback">
						<p>{{ item.filename }}</p>
						<a :href="item.url" target="_blank" rel="noopener">Open file</a>
					</div>
				</div>

				<div class="media-preview-sidebar">
					<h3>{{ item.filename }}</h3>
					<a :href="item.url" target="_blank" rel="noopener" class="open-file-link">Open original</a>

					<h4>Used in posts</h4>
					<p v-if="loadingPosts" class="muted">Loading posts…</p>
					<p v-else-if="postsError" class="inline-notification error">{{ postsError }}</p>
					<p v-else-if="!posts.length" class="inline-notification note">
						This media has not been used in any posts yet.
					</p>
					<ul v-else class="post-usage-list">
						<li v-for="post in posts" :key="post.id">
							<router-link :to="{ name: 'postDetails', params: { id: post.id } }" @click="close">
								<span class="post-usage-meta">
									<Icon v-if="post.socialAccountIcon" :icon="post.socialAccountIcon" />
									{{ post.socialAccountIdentity || 'Unknown account' }}
									<span class="post-usage-state">{{ post.state }}</span>
								</span>
								<span class="post-usage-content">{{ post.content }}</span>
								<span class="post-usage-date">{{ post.postedDate || post.created }}</span>
							</router-link>
						</li>
					</ul>

					<DangerZone
						description="Permanently removes this file from the media library. Posts that used it will no longer reference this attachment."
					>
						<p v-if="deleteError" class="inline-notification error">{{ deleteError }}</p>
						<button
							type="button"
							class="inline-icon bad"
							:disabled="deleting"
							@click="deleteMedia"
						>
							<Icon icon="mdi:delete-outline" width="1em" height="1em" aria-hidden="true" />
							<span>{{ deleting ? 'Deleting…' : 'Delete media file' }}</span>
						</button>
					</DangerZone>

					<div class="sidebar-actions">
						<button type="button" class="good" @click="useInNewPost">Use in new post</button>
						<button type="button" class="neutral" @click="close">Close</button>
					</div>
				</div>
			</div>
		</div>
	</div>
	</Teleport>
</template>

<script setup>
	import { ref, watch, onMounted, onUnmounted } from 'vue';
	import { useRouter } from 'vue-router';
	import { Icon } from '@iconify/vue';
	import { waitForClient } from '../javascript/util';
	import DangerZone from './DangerZone.vue';

	const props = defineProps({
		open: {
			type: Boolean,
			default: false,
		},
		item: {
			type: Object,
			default: null,
		},
	});

	const emit = defineEmits(['close', 'deleted']);

	const router = useRouter();
	const posts = ref([]);
	const loadingPosts = ref(false);
	const postsError = ref('');
	const deleting = ref(false);
	const deleteError = ref('');

	function isImage(filename) {
		const ext = (filename || '').split('.').pop()?.toLowerCase();
		return ['jpg', 'jpeg', 'png', 'gif', 'webp', 'svg', 'bmp', 'ico', 'apng', 'tiff'].includes(ext);
	}

	function isVideo(filename) {
		const ext = (filename || '').split('.').pop()?.toLowerCase();
		return ['mp4', 'webm', 'ogg', 'mov', 'avi', 'mkv'].includes(ext);
	}

	function close() {
		emit('close');
	}

	async function loadPosts() {
		if (!props.item?.filename) {
			posts.value = [];
			return;
		}

		loadingPosts.value = true;
		postsError.value = '';
		try {
			await waitForClient();
			const res = await window.client.getMediaPosts({ filename: props.item.filename });
			posts.value = res.posts ?? [];
		} catch (e) {
			postsError.value = e.message || 'Failed to load posts for this media';
			posts.value = [];
		} finally {
			loadingPosts.value = false;
		}
	}

	function useInNewPost() {
		if (!props.item?.url) {
			return;
		}
		close();
		router.push({
			name: 'postBox',
			query: { mediaUrl: props.item.url },
		});
	}

	async function deleteMedia() {
		if (!props.item?.filename || deleting.value) {
			return;
		}

		const postWarning = posts.value.length > 0
			? `\n\nThis file is attached to ${posts.value.length} post(s). Those posts will lose this attachment.`
			: '';

		if (!confirm(`Delete "${props.item.filename}" permanently? This cannot be undone.${postWarning}`)) {
			return;
		}

		deleting.value = true;
		deleteError.value = '';
		try {
			await waitForClient();
			await window.client.deleteMedia({ filename: props.item.filename });
			emit('deleted');
			close();
			window.dispatchEvent(new CustomEvent('media-deleted'));
		} catch (e) {
			deleteError.value = e.message || 'Failed to delete media file';
		} finally {
			deleting.value = false;
		}
	}

	watch(
		() => [props.open, props.item?.filename],
		([isOpen]) => {
			if (isOpen) {
				loadPosts();
			} else {
				posts.value = [];
				postsError.value = '';
				deleteError.value = '';
			}
		},
		{ immediate: true },
	);

	function onKeydown(event) {
		if (event.key === 'Escape' && props.open) {
			close();
		}
	}

	onMounted(() => {
		document.addEventListener('keydown', onKeydown);
	});

	onUnmounted(() => {
		document.removeEventListener('keydown', onKeydown);
	});
</script>

<style scoped>
	.modal-overlay {
		position: fixed;
		top: 0;
		left: 0;
		width: 100%;
		height: 100%;
		background-color: rgba(0, 0, 0, 0.65);
		display: flex;
		justify-content: center;
		align-items: center;
		z-index: 10000;
		padding: 1rem;
		box-sizing: border-box;
	}

	.media-preview-modal {
		position: relative;
		background: var(--background-primary, #fff);
		border-radius: 0.75rem;
		box-shadow: 0 8px 24px rgba(0, 0, 0, 0.25);
		width: min(1100px, 100%);
		max-height: calc(100vh - 2rem);
		overflow: hidden;
	}

	.close-button {
		position: absolute;
		top: 0.75rem;
		right: 0.75rem;
		z-index: 2;
		width: 2rem;
		height: 2rem;
		padding: 0;
		border-radius: 50%;
		font-size: 1.4rem;
		line-height: 1;
	}

	.media-preview-layout {
		display: grid;
		grid-template-columns: minmax(0, 1.4fr) minmax(280px, 0.9fr);
		min-height: 420px;
		max-height: calc(100vh - 2rem);
	}

	.media-preview-main {
		display: flex;
		align-items: center;
		justify-content: center;
		background: #111;
		min-height: 320px;
		overflow: hidden;
	}

	.media-preview-image,
	.media-preview-video {
		max-width: 100%;
		max-height: calc(100vh - 2rem);
		object-fit: contain;
		display: block;
	}

	.media-preview-fallback {
		color: #fff;
		text-align: center;
		padding: 2rem;
	}

	.media-preview-sidebar {
		position: relative;
		display: flex;
		flex-direction: column;
		gap: 0.75rem;
		padding: 1.25rem;
		border-left: 1px solid var(--border-color, #ddd);
		overflow-y: auto;
		background: var(--background-primary, #fff);
		color: var(--text-primary, #222);
		min-width: 0;
	}

	.media-preview-sidebar h3,
	.media-preview-sidebar h4 {
		margin: 0;
	}

	.media-preview-sidebar h3 {
		word-break: break-all;
		padding-right: 2rem;
		font-size: 1rem;
	}

	.media-preview-sidebar h4 {
		font-size: 0.95rem;
		margin-top: 0.5rem;
	}

	.open-file-link {
		font-size: 0.9rem;
	}

	.post-usage-list {
		list-style: none;
		margin: 0;
		padding: 0;
		display: flex;
		flex-direction: column;
		gap: 0.5rem;
		flex: 1;
		overflow-y: auto;
	}

	.post-usage-list a {
		display: block;
		padding: 0.65rem 0.75rem;
		border: 1px solid var(--border-color, #ddd);
		border-radius: 0.5rem;
		text-decoration: none;
		color: inherit;
		background: var(--card-bg, #f8f8f8);
	}

	.post-usage-list a:hover {
		border-color: var(--accent-color, #5a9fd4);
	}

	.post-usage-meta {
		display: flex;
		align-items: center;
		gap: 0.35rem;
		font-size: 0.85rem;
		font-weight: 600;
		margin-bottom: 0.25rem;
	}

	.post-usage-state {
		margin-left: auto;
		font-size: 0.75rem;
		font-weight: 500;
		opacity: 0.8;
		text-transform: capitalize;
	}

	.post-usage-content {
		display: -webkit-box;
		-webkit-line-clamp: 2;
		-webkit-box-orient: vertical;
		overflow: hidden;
		font-size: 0.9rem;
		line-height: 1.35;
	}

	.post-usage-date {
		display: block;
		margin-top: 0.35rem;
		font-size: 0.75rem;
		opacity: 0.75;
	}

	.sidebar-actions {
		display: flex;
		flex-wrap: wrap;
		gap: 0.5rem;
		margin-top: auto;
		padding-top: 0.5rem;
	}

	.media-preview-sidebar :deep(.danger-zone) {
		margin-top: 0.5rem;
	}

	.muted {
		opacity: 0.7;
	}

	@media (max-width: 800px) {
		.media-preview-layout {
			grid-template-columns: 1fr;
			max-height: calc(100vh - 2rem);
			overflow-y: auto;
		}

		.media-preview-sidebar {
			border-left: none;
			border-top: 1px solid var(--border-color, #ddd);
			max-height: 40vh;
		}
	}
</style>
