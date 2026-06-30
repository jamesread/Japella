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
			<div v-for="item in items" :key="item.filename" class="media-item">
				<a :href="item.url" target="_blank" rel="noopener" class="media-link">
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
				</a>
				<p class="media-filename">{{ item.filename }}</p>
			</div>
		</div>
	</Section>
</template>

<script setup>
	import { ref, onMounted, onUnmounted } from 'vue';
	import { Icon } from '@iconify/vue';
	import { waitForClient } from '../javascript/util';
	import Section from 'picocrank/vue/components/Section.vue';

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

	function onMediaUploaded() {
		loadList();
	}

	onMounted(() => {
		loadList();
		window.addEventListener('media-uploaded', onMediaUploaded);
	});

	onUnmounted(() => {
		window.removeEventListener('media-uploaded', onMediaUploaded);
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
		background: var(--card-bg, #f5f5f5);
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
	}

	.media-filename {
		margin: 0;
		padding: 0.4em 0.5em;
		font-size: 0.8em;
		word-break: break-all;
		text-align: center;
	}

	.muted {
		opacity: 0.7;
	}
</style>
