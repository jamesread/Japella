<template>
	<MediaGallery @select="openPreview" />

	<MediaUpload />

	<MediaPreviewDialog :open="previewOpen" :item="previewItem" @close="closePreview" @deleted="closePreview" />
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

	function closePreview() {
		previewOpen.value = false;
		previewItem.value = null;
	}
</script>
