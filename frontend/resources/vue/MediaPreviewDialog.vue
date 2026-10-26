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
					<p v-if="sizeBytes || suitability?.width" class="muted media-size">
						<span v-if="sizeBytes" :title="`${Number(sizeBytes).toLocaleString()} bytes`">{{ formatFileSize(sizeBytes) }}</span>
						<span v-if="suitability?.width"> · {{ suitability.width }}×{{ suitability.height }}</span>
					</p>
					<a :href="item.url" target="_blank" rel="noopener" class="open-file-link">Open original</a>

					<h4>Suitability</h4>
					<p v-if="loadingSuitability" class="muted">Checking…</p>
					<NotificationBlock v-else-if="suitabilityError" type="error" :message="suitabilityError" />
					<NotificationBlock
						v-else-if="suitability && !suitability.networks.length"
						type="note"
						message="No active social accounts on networks with known media limits."
					/>
					<ul v-else-if="suitability" class="suitability-list">
						<li v-for="network in suitability.networks" :key="network.protocol" :class="`suitability--${network.verdict}`">
							<div class="suitability-network">
								<Icon :icon="network.icon" aria-hidden="true" />
								<span class="suitability-name">{{ network.name }}</span>
								<span class="tag suitability-verdict" :class="VERDICT_KARMA[network.verdict]">
									<Icon :icon="VERDICT_ICONS[network.verdict]" aria-hidden="true" />
									{{ VERDICT_LABELS[network.verdict] }}
								</span>
							</div>
							<ul class="suitability-checks">
								<li v-for="(check, i) in network.checks" :key="i" :class="`suitability-check--${check.level}`">
									<Icon :icon="VERDICT_ICONS[check.level]" aria-hidden="true" />
									<span>{{ check.message }}</span>
								</li>
							</ul>
						</li>
					</ul>
					<p v-if="suitability?.networks.length" class="muted suitability-note">
						Based on each network's published limits; Mastodon instances can set their own.
					</p>

					<h4>Optimised copies</h4>
					<div class="media-optimise">
						<template v-if="item.optimisedOf">
							<p class="muted">
								{{ item.optimisedKind === 'posting'
									? 'This copy was resized and re-saved at high quality to fit your networks\' limits.'
									: 'This lossless copy looks identical to the original but is smaller.' }}
							</p>
							<button type="button" class="neutral small" @click="openOther(item.optimisedOf)">Open original</button>
						</template>
						<template v-else>
							<div v-for="mode in OPTIMISE_MODES" :key="mode.id" class="media-optimise-mode">
								<strong>{{ mode.title }}</strong>
								<p class="muted">{{ results[mode.id] || mode.description }}</p>
								<button
									v-if="versions[mode.id]"
									type="button"
									class="good small"
									@click="openOther(versions[mode.id])"
								>
									Open {{ mode.short }}
								</button>
								<button
									v-else-if="mode.supports(item.filename)"
									type="button"
									class="neutral small"
									:disabled="Boolean(optimising)"
									@click="optimise(mode.id)"
								>
									<Icon :icon="mode.icon" aria-hidden="true" />
									<span>{{ optimising === mode.id ? 'Working…' : `Create ${mode.short}` }}</span>
								</button>
								<p v-else class="muted">{{ mode.unsupported }}</p>
							</div>
						</template>
						<NotificationBlock v-if="optimiseError" type="error" :message="optimiseError" />
					</div>

					<h4>Used in posts</h4>
					<p v-if="loadingPosts" class="muted">Loading posts…</p>
					<NotificationBlock v-else-if="postsError" type="error" :message="postsError" />
					<NotificationBlock
						v-else-if="!posts.length"
						type="note"
						message="This media has not been used in any posts yet."
					/>
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
						title="Danger zone"
						subtitle="Remove this media file"
						description="Show destructive actions"
						warning="Permanently removes this file from the media library. Posts that used it will no longer reference this attachment."
					>
						<NotificationBlock v-if="deleteError" type="error" :message="deleteError" />
						<div role="toolbar" class="danger-zone-actions">
							<button
								type="button"
								class="inline-icon bad"
								:disabled="deleting"
								@click="deleteMedia"
							>
								<Icon icon="mdi:delete-outline" width="1em" height="1em" aria-hidden="true" />
								<span>{{ deleting ? 'Deleting…' : 'Delete media file' }}</span>
							</button>
						</div>
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
	import { ref, computed, watch, onMounted, onUnmounted } from 'vue';
	import { useRouter } from 'vue-router';
	import { Icon } from '@iconify/vue';
	import { waitForClient, formatFileSize } from '../javascript/util';
	import { VERDICT_ICONS, VERDICT_LABELS, VERDICT_KARMA } from '../javascript/mediaSuitability.js';
	import DangerZone from 'picocrank/vue/components/DangerZone.vue';
	import NotificationBlock from 'picocrank/vue/components/NotificationBlock.vue';

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

	const emit = defineEmits(['close', 'deleted', 'open-item']);

	const router = useRouter();
	const posts = ref([]);
	const loadingPosts = ref(false);
	const postsError = ref('');
	const deleting = ref(false);
	const deleteError = ref('');
	const suitability = ref(null);
	const loadingSuitability = ref(false);
	const suitabilityError = ref('');
	const OPTIMISE_MODES = [
		{
			id: 'posting',
			title: 'Copy for posting',
			short: 'copy for posting',
			icon: 'mdi:image-size-select-large',
			description: 'Resized and re-saved at high quality to fit every network you post to. Usually looks the same, and is much smaller for photos.',
			supports: (name) => /\.(png|jpe?g|webp|bmp|tiff?)$/i.test(name),
			unsupported: 'Only still images can be prepared for posting.',
		},
		{
			id: 'lossless',
			title: 'Lossless copy',
			short: 'lossless copy',
			icon: 'mdi:image-auto-adjust',
			description: 'Identical pixels, smaller file. Works well for screenshots and graphics; photos only shrink a little.',
			supports: (name) => /\.(png|jpe?g)$/i.test(name),
			unsupported: 'Only PNG and JPEG images have a lossless copy.',
		},
	];

	// The mode being run, or ''.
	const optimising = ref('');
	const optimiseError = ref('');
	// Per mode: the result message, and copies made here before the list reloads.
	const results = ref({});
	const created = ref({});

	const sizeBytes = computed(() => props.item?.sizeBytes || suitability.value?.sizeBytes || 0);
	const versions = computed(() => ({
		posting: props.item?.postingVersion || created.value.posting || '',
		lossless: props.item?.optimisedVersion || created.value.lossless || '',
	}));

	function openOther(filename) {
		emit('open-item', { filename, url: `/media/files/${filename}` });
	}

	async function optimise(mode) {
		optimising.value = mode;
		optimiseError.value = '';
		try {
			await waitForClient();
			const res = await window.client.optimiseMedia({ filename: props.item.filename, mode });
			if (res.item) {
				const saved = Number(res.originalSizeBytes) - Number(res.optimisedSizeBytes);
				const pct = Math.round((saved / Number(res.originalSizeBytes)) * 100);
				created.value = { ...created.value, [mode]: res.item.filename };
				results.value = {
					...results.value,
					[mode]: `${formatFileSize(res.optimisedSizeBytes)} instead of ${formatFileSize(res.originalSizeBytes)} (${pct}% smaller). ${res.message}`,
				};
				window.dispatchEvent(new CustomEvent('media-uploaded'));
			} else {
				results.value = { ...results.value, [mode]: res.message };
			}
		} catch (e) {
			optimiseError.value = e.rawMessage || e.message || 'Failed to optimise this image';
		} finally {
			optimising.value = '';
		}
	}


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

	async function loadSuitability() {
		const filename = props.item?.filename;
		suitability.value = null;
		suitabilityError.value = '';
		if (!filename) {
			return;
		}

		loadingSuitability.value = true;
		try {
			await waitForClient();
			const res = await window.client.getMediaSuitability({ filename });
			// Ignore a slow answer for a file that's no longer shown.
			if (props.item?.filename === filename) {
				suitability.value = res;
			}
		} catch (e) {
			suitabilityError.value = e.rawMessage || e.message || 'Failed to check this media';
		} finally {
			loadingSuitability.value = false;
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
			created.value = {};
			results.value = {};
			optimiseError.value = '';
			if (isOpen) {
				loadPosts();
				loadSuitability();
			} else {
				suitability.value = null;
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

	.suitability-list {
		list-style: none;
		margin: 0;
		padding: 0;
		display: flex;
		flex-direction: column;
		gap: 0.5rem;
	}

	.suitability-network {
		display: flex;
		align-items: center;
		gap: 0.35rem;
		font-weight: 600;
	}

	.suitability-name {
		flex: 1;
	}

	.suitability-verdict {
		display: inline-flex;
		align-items: center;
		gap: 0.2rem;
		font-size: 0.75rem;
	}

	.suitability-checks {
		list-style: none;
		margin: 0.2rem 0 0;
		padding: 0 0 0 1.4rem;
		font-size: 0.85rem;
	}

	.suitability-checks li {
		display: flex;
		align-items: flex-start;
		gap: 0.3rem;
	}

	.suitability-checks svg {
		flex-shrink: 0;
		margin-top: 0.15em;
	}

	.suitability-check--ok {
		opacity: 0.75;
	}

	.suitability-note {
		font-size: 0.8rem;
		margin: 0.4rem 0 0;
	}

	.media-optimise {
		display: flex;
		flex-direction: column;
		align-items: flex-start;
		gap: 0.4rem;
	}

	.media-optimise-mode {
		display: flex;
		flex-direction: column;
		align-items: flex-start;
		gap: 0.25rem;
		width: 100%;
	}

	.media-optimise-mode + .media-optimise-mode {
		padding-top: 0.5rem;
		border-top: 1px solid var(--border-color, rgba(127, 127, 127, 0.25));
	}

	.media-optimise p {
		margin: 0;
		font-size: 0.85rem;
	}

	.media-optimise button {
		display: inline-flex;
		align-items: center;
		gap: 0.3rem;
	}

	.media-size {
		margin: -0.25rem 0 0.5rem;
		font-size: 0.85rem;
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

	.danger-zone-actions {
		display: flex;
		flex-wrap: wrap;
		gap: 0.5rem;
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
