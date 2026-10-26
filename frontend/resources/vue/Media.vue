<template>
	<MediaGallery @select="openPreview" />

	<MediaUpload />

	<MediaPreviewDialog
		:open="previewOpen"
		:item="previewItem"
		@close="closePreview"
		@deleted="closePreview"
		@open-item="openRelated"
	/>
</template>

<script setup>
	import { ref } from 'vue';
	import MediaGallery from './MediaGallery.vue';
	import MediaUpload from './MediaUpload.vue';
	import MediaPreviewDialog from './MediaPreviewDialog.vue';

	const previewOpen = ref(false);
	const previewItem = ref(null);

	function openPreview(item) {
		previewItem.value = item;
		previewOpen.value = true;
	}

	// Opens an original or copy from the dialog, with its full details (links
	// between copies, size, suitability) from a fresh listing.
	async function openRelated(item) {
		openPreview(item);
		try {
			const res = await window.client.listMedia({});
			const full = (res.items || []).find((i) => i.filename === item.filename);
			if (full && previewItem.value?.filename === item.filename) {
				previewItem.value = full;
			}
		} catch {
			// Keep the basic details.
		}
	}

	function closePreview() {
		previewOpen.value = false;
		previewItem.value = null;
	}
</script>
