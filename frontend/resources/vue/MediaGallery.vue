<template>
	<Section
		title="Media Gallery"
		subtitle="Browse uploaded images and videos in your media library."
		:icon="ImageIcon"
		classes="media-gallery"
	>
		<template #toolbar>
			<button
				type="button"
				class="neutral"
				:title="needsPosting.length
					? `Create copies for posting of ${needsPosting.length} images that don't suit all your networks; originals are kept`
					: 'Every image already suits your networks or has a copy for posting'"
				:disabled="loading || Boolean(optimisingAll) || !needsPosting.length"
				@click="optimiseAll('posting', needsPosting)"
			>
				<Icon icon="mdi:image-size-select-large" />
				<span>{{ optimisingAll === 'posting' ? `Preparing ${optimiseProgress}/${optimiseTotal}…` : 'Prepare for posting' }}</span>
			</button>
			<button
				type="button"
				class="neutral"
				:title="optimisable.length ? `Create lossless copies of ${optimisable.length} PNG/JPEG images; originals are kept` : 'Every PNG and JPEG image already has a lossless copy'"
				:disabled="loading || Boolean(optimisingAll) || !optimisable.length"
				@click="optimiseAll('lossless', optimisable)"
			>
				<Icon icon="mdi:image-auto-adjust" />
				<span>{{ optimisingAll === 'lossless' ? `Optimising ${optimiseProgress}/${optimiseTotal}…` : 'Optimise all (lossless)' }}</span>
			</button>
			<button type="button" class="neutral" title="Refresh" :disabled="loading" @click="loadList">
				<Icon icon="material-symbols:refresh" />
			</button>
		</template>

		<NotificationBlock v-if="optimiseSummary" type="note" :message="optimiseSummary" />
		<p v-if="loading" class="muted">Loading…</p>
		<NotificationBlock v-else-if="error" type="error" :message="error" />
		<NotificationBlock
			v-else-if="!items.length"
			type="note"
			message="No media files yet. Upload some from the form below."
		/>

		<div v-else class="media-grid">
			<button
				v-for="item in items"
				:key="item.filename"
				type="button"
				class="media-item"
				@click.prevent.stop="openPreview(item)"
			>
				<span class="media-link">
					<img
						v-if="isImage(item.filename)"
						:src="item.url"
						:alt="item.filename"
						class="media-thumb"
						loading="lazy"
					/>
					<video
						v-else-if="isVideo(item.filename)"
						:src="item.url"
						class="media-thumb"
						preload="metadata"
						muted
						playsinline
					/>
					<div v-else class="media-placeholder">
						<span>{{ item.filename }}</span>
					</div>
				</span>
				<span class="media-filename">{{ item.filename }}</span>
				<span
					v-if="item.optimisedOf"
					class="media-optimised"
					:title="item.optimisedKind === 'posting' ? `Resized and re-saved for posting, from ${item.optimisedOf}` : `A lossless copy of ${item.optimisedOf}`"
				>
					<Icon :icon="item.optimisedKind === 'posting' ? 'mdi:image-size-select-large' : 'mdi:image-auto-adjust'" aria-hidden="true" />
					{{ item.optimisedKind === 'posting' ? 'For posting' : 'Lossless copy' }}
				</span>
				<MediaSuitabilityBadges :networks="item.suitability || []" class="media-suitability" />
			</button>
		</div>
	</Section>
</template>

<script setup>
	import { ref, computed, onMounted, onUnmounted } from 'vue';
	import { Icon } from '@iconify/vue';
	import { ImageIcon } from '@hugeicons/core-free-icons';
	import { waitForClient, formatFileSize } from '../javascript/util';
	import Section from 'picocrank/vue/components/Section.vue';
	import NotificationBlock from 'picocrank/vue/components/NotificationBlock.vue';
	import MediaSuitabilityBadges from './MediaSuitabilityBadges.vue';

	const emit = defineEmits(['select']);

	const items = ref([]);
	const loading = ref(true);
	const error = ref('');
	// The mode being run for the whole library, or ''.
	const optimisingAll = ref('');
	const optimiseProgress = ref(0);
	const optimiseTotal = ref(0);
	const optimiseSummary = ref('');

	// Originals that could still get a lossless copy.
	const optimisable = computed(() => items.value.filter((item) => !item.optimisedOf
		&& !item.optimisedVersion
		&& /\.(png|jpe?g)$/i.test(item.filename)));

	// Still images that don't suit every network and have no copy for posting.
	const needsPosting = computed(() => items.value.filter((item) => !item.optimisedOf
		&& !item.postingVersion
		&& /\.(png|jpe?g|webp|bmp|tiff?)$/i.test(item.filename)
		&& (item.suitability || []).some((n) => n.verdict !== 'ok')));

	// One at a time, so a large library doesn't load the server all at once.
	async function optimiseAll(mode, candidates) {
		const todo = [...candidates];
		optimisingAll.value = mode;
		optimiseProgress.value = 0;
		optimiseTotal.value = todo.length;
		optimiseSummary.value = '';
		let created = 0;
		let saved = 0;
		let failed = 0;
		for (const item of todo) {
			try {
				const res = await window.client.optimiseMedia({ filename: item.filename, mode });
				if (res.item) {
					created += 1;
					saved += Number(res.originalSizeBytes) - Number(res.optimisedSizeBytes);
				}
			} catch {
				failed += 1;
			}
			optimiseProgress.value += 1;
		}
		optimisingAll.value = '';
		const unchanged = todo.length - created - failed;
		const kind = mode === 'posting' ? 'for posting' : 'lossless';
		optimiseSummary.value = `Created ${created} ${kind} ${created === 1 ? 'copy' : 'copies'}, saving ${formatFileSize(saved)}.`
			+ (unchanged ? ` ${unchanged} couldn't be made smaller.` : '')
			+ (failed ? ` ${failed} couldn't be optimised.` : '');
		loadList();
	}

	function isImage(filename) {
		const ext = (filename || '').split('.').pop()?.toLowerCase();
		return ['jpg', 'jpeg', 'png', 'gif', 'webp', 'svg', 'bmp', 'ico', 'apng', 'tiff'].includes(ext);
	}

	function isVideo(filename) {
		const ext = (filename || '').split('.').pop()?.toLowerCase();
		return ['mp4', 'webm', 'ogg', 'mov', 'avi', 'mkv'].includes(ext);
	}

	async function loadList() {
		loading.value = true;
		error.value = '';
		try {
			await waitForClient();
			const res = await window.client.listMedia({});
			items.value = res.items ?? [];
		} catch (e) {
			error.value = e.message || 'Failed to load media list';
			items.value = [];
		} finally {
			loading.value = false;
		}
	}

	function openPreview(item) {
		emit('select', item);
	}

	function onMediaUploaded() {
		loadList();
	}

	function onMediaDeleted() {
		loadList();
	}

	onMounted(() => {
		loadList();
		window.addEventListener('media-uploaded', onMediaUploaded);
		window.addEventListener('media-deleted', onMediaDeleted);
	});

	onUnmounted(() => {
		window.removeEventListener('media-uploaded', onMediaUploaded);
		window.removeEventListener('media-deleted', onMediaDeleted);
	});
</script>

<style scoped>
	.media-grid {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
		gap: 1em;
		margin-top: 0.25em;
	}

	.media-item {
		display: flex;
		flex-direction: column;
		align-items: center;
		border: 1px solid var(--border-color, #ccc);
		border-radius: 0.5em;
		overflow: hidden;
		background: var(--standout-bg-color, #f5f5f5);
		padding: 0;
		cursor: pointer;
		text-align: inherit;
		font: inherit;
		appearance: none;
	}

	.media-item:hover {
		border-color: var(--accent-color, #5a9fd4);
		box-shadow: 0 2px 8px rgba(0, 0, 0, 0.12);
	}

	.media-link {
		display: block;
		width: 100%;
		aspect-ratio: 1;
		overflow: hidden;
	}

	.media-thumb {
		width: 100%;
		height: 100%;
		object-fit: cover;
		display: block;
		pointer-events: none;
		user-select: none;
		-webkit-user-drag: none;
	}

	.media-placeholder {
		width: 100%;
		height: 100%;
		display: flex;
		align-items: center;
		justify-content: center;
		padding: 0.5em;
		text-align: center;
		font-size: 0.85em;
		word-break: break-all;
		color: var(--text-secondary, #666);
	}

	.media-filename {
		margin: 0;
		padding: 0.4em 0.5em;
		font-size: 0.8em;
		word-break: break-all;
		text-align: center;
		color: var(--text-primary, #333);
	}

	html[data-theme="dark"] {
		.media-placeholder {
			color: #aaa;
		}

		.media-filename {
			color: #ddd;
		}
	}

	.muted {
		opacity: 0.7;
	}

	.media-optimised {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		gap: 0.2em;
		font-size: 0.75em;
		padding-bottom: 0.3em;
		opacity: 0.8;
	}

	.media-suitability {
		justify-content: center;
		padding: 0 0.5em 0.4em;
	}
</style>
