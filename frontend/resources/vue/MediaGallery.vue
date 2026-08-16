<template>
	<Section
		title="Media Gallery"
		subtitle="Browse uploaded images and videos in your media library."
		classes="media-gallery"
	>
		<template #toolbar>
			<button type="button" class="neutral" title="Refresh" :disabled="loading" @click="loadList">
				<Icon icon="material-symbols:refresh" />
			</button>
		</template>

		<p v-if="loading" class="muted">Loading…</p>
		<p v-else-if="error" class="inline-notification error">{{ error }}</p>
		<p v-else-if="!items.length" class="inline-notification note">
			No media files yet. Upload some from the form below.
		</p>

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
			</button>
		</div>
	</Section>
</template>

<script setup>
	import { ref, onMounted, onUnmounted } from 'vue';
	import { Icon } from '@iconify/vue';
	import { waitForClient } from '../javascript/util';
	import Section from 'picocrank/vue/components/Section.vue';

	const emit = defineEmits(['select']);

	const items = ref([]);
	const loading = ref(true);
	const error = ref('');

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
</style>
