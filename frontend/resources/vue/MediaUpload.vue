<template>
	<Section
		title="Media Upload"
		subtitle="Upload images and videos to your media library. You can also paste images from your clipboard (Ctrl+V)."
		classes="media-upload"
	>
		<form id="media-upload-form" @submit.prevent="submitForm">
			<p v-if="formMessage" class="inline-notification" :class="formMessageType">{{ formMessage }}</p>

			<fieldset class="upload-fieldset">
				<div ref="previewEl" class="image-preview">
					<div
						v-for="item in uploadItems"
						:key="item.id"
						class="uploaded-file"
						:class="`uploaded-file--${item.status}`"
					>
						<div class="upload-preview">
							<img v-if="item.objectUrl" :src="item.objectUrl" :alt="item.name" />
							<div v-else class="upload-placeholder" aria-hidden="true" />
							<div v-if="item.status === 'uploading'" class="upload-overlay">
								<span>{{ item.progress }}%</span>
							</div>
							<div v-else-if="item.status === 'success'" class="upload-overlay upload-overlay--success">
								<span>Uploaded</span>
							</div>
						</div>
						<p class="upload-name"><small>{{ item.name }}</small></p>
						<p><small>{{ returnFileSize(item.size) }}</small></p>
						<progress
							v-if="item.status === 'uploading'"
							class="upload-progress"
							:value="item.progress"
							max="100"
						>
							{{ item.progress }}%
						</progress>
						<p v-else-if="item.status === 'error'" class="upload-status error">
							{{ item.error }}
						</p>
					</div>
				</div>
				<label for="media-file" class="drag-drop">
					{{ t('section.media.upload.label') }}:
					<input
						id="media-file"
						ref="fileupload"
						type="file"
						name="media-file"
						multiple
						accept="image/*,video/*"
					/>
				</label>
			</fieldset>
			<fieldset>
				<button id="submit" type="submit" :disabled="uploading || !uploadItems.length">
					{{ uploading ? t('section.media.upload.submitting') : t('section.media.upload.submit') }}
				</button>
			</fieldset>
		</form>
	</Section>
</template>

<script setup>
	import { ref, onMounted, onUnmounted } from 'vue';
	import { useI18n } from 'vue-i18n';
	import Section from 'picocrank/vue/components/Section.vue';

	const { t } = useI18n();

	const fileupload = ref(null);
	const previewEl = ref(null);
	const uploadItems = ref([]);
	const uploading = ref(false);
	const formMessage = ref('');
	const formMessageType = ref('');
	let nextItemId = 0;

	const fileTypes = [
		'image/apng',
		'image/bmp',
		'image/gif',
		'image/jpeg',
		'image/pjpeg',
		'image/png',
		'image/svg+xml',
		'image/tiff',
		'image/webp',
		'image/x-icon',
	];

	function validFileType(file) {
		return fileTypes.includes(file.type);
	}

	function returnFileSize(number) {
		if (number < 1e3) {
			return `${number} bytes`;
		}
		if (number >= 1e3 && number < 1e6) {
			return `${(number / 1e3).toFixed(1)} KB`;
		}
		return `${(number / 1e6).toFixed(1)} MB`;
	}

	function revokeItemUrl(item) {
		if (item.objectUrl) {
			URL.revokeObjectURL(item.objectUrl);
		}
	}

	function clearUploadItems() {
		for (const item of uploadItems.value) {
			revokeItemUrl(item);
		}
		uploadItems.value = [];
	}

	function createUploadItem(file) {
		return {
			id: nextItemId++,
			file,
			name: file.name,
			size: file.size,
			objectUrl: URL.createObjectURL(file),
			status: 'pending',
			progress: 0,
			error: '',
		};
	}

	function updateImageDisplay() {
		const input = fileupload.value;
		if (!input) {
			return;
		}

		clearUploadItems();
		formMessage.value = '';
		formMessageType.value = '';

		for (const file of input.files) {
			if (validFileType(file)) {
				uploadItems.value.push(createUploadItem(file));
			}
		}
	}

	function uploadFile(item) {
		return new Promise((resolve, reject) => {
			const xhr = new XMLHttpRequest();
			const formData = new FormData();
			formData.append('file', item.file);

			xhr.upload.addEventListener('progress', (event) => {
				if (event.lengthComputable) {
					item.progress = Math.round((event.loaded / event.total) * 100);
				}
			});

			xhr.addEventListener('load', () => {
				if (xhr.status >= 200 && xhr.status < 300) {
					resolve();
					return;
				}
				reject(new Error(xhr.responseText || xhr.statusText || `HTTP ${xhr.status}`));
			});

			xhr.addEventListener('error', () => {
				reject(new Error('Network error'));
			});

			xhr.open('POST', '/upload');
			xhr.withCredentials = true;
			xhr.send(formData);
		});
	}

	async function submitForm() {
		const input = fileupload.value;
		const pendingItems = uploadItems.value.filter((item) => item.status === 'pending' || item.status === 'error');

		if (!pendingItems.length) {
			formMessage.value = 'No files currently selected for upload';
			formMessageType.value = 'note';
			return;
		}

		formMessage.value = '';
		formMessageType.value = '';
		uploading.value = true;

		let successCount = 0;
		let hadUnsupported = false;

		const uploadTasks = pendingItems.map(async (item) => {
			if (!validFileType(item.file)) {
				item.status = 'error';
				item.error = `File type ${item.file.type} is not supported.`;
				hadUnsupported = true;
				return;
			}

			item.status = 'uploading';
			item.progress = 0;
			item.error = '';

			try {
				await uploadFile(item);
				item.status = 'success';
				item.progress = 100;
				successCount += 1;
			} catch (error) {
				item.status = 'error';
				item.error = error.message || 'Upload failed';
			}
		});

		await Promise.all(uploadTasks);

		uploading.value = false;

		if (successCount > 0) {
			window.dispatchEvent(new CustomEvent('media-uploaded'));
		}

		if (successCount === pendingItems.length) {
			formMessage.value =
				successCount === 1 ? '1 file uploaded successfully.' : `${successCount} files uploaded successfully.`;
			formMessageType.value = 'good';
			input.value = '';
		} else if (successCount > 0) {
			formMessage.value = `${successCount} of ${pendingItems.length} files uploaded successfully.`;
			formMessageType.value = 'note';
		} else if (hadUnsupported) {
			formMessage.value = 'Some files could not be uploaded because their type is not supported.';
			formMessageType.value = 'error';
		} else {
			formMessage.value = 'Upload failed for all selected files.';
			formMessageType.value = 'error';
		}
	}

	onMounted(() => {
		const fileInput = fileupload.value;
		if (!fileInput) {
			return;
		}

		fileInput.addEventListener('dragover', (event) => {
			event.preventDefault();
			event.stopPropagation();
			fileInput.classList.add('drag-over');
		});

		fileInput.addEventListener('dragleave', (event) => {
			event.preventDefault();
			event.stopPropagation();
			fileInput.classList.remove('drag-over');
		});

		fileInput.addEventListener('drop', (event) => {
			event.preventDefault();
			event.stopPropagation();
			fileInput.classList.remove('drag-over');
			if (event.dataTransfer.files.length > 0) {
				fileInput.files = event.dataTransfer.files;
				updateImageDisplay();
			}
		});

		fileInput.addEventListener('change', updateImageDisplay);

		const handlePaste = (event) => {
			const items = event.clipboardData?.items;
			if (!items) {
				return;
			}

			const pastedImages = [];
			for (const item of items) {
				if (item.type.startsWith('image/')) {
					const file = item.getAsFile();
					if (file) {
						pastedImages.push(file);
					}
				}
			}
			if (pastedImages.length === 0) {
				return;
			}

			event.preventDefault();
			const dt = new DataTransfer();
			for (const f of fileInput.files) {
				dt.items.add(f);
			}
			for (const f of pastedImages) {
				dt.items.add(f);
			}
			fileInput.files = dt.files;
			updateImageDisplay();
		};

		document.addEventListener('paste', handlePaste);
		fileInput._pasteHandler = handlePaste;
	});

	onUnmounted(() => {
		const fileInput = fileupload.value;
		if (fileInput?._pasteHandler) {
			document.removeEventListener('paste', fileInput._pasteHandler);
		}

		clearUploadItems();
	});
</script>

<style scoped>
	.upload-fieldset {
		flex-direction: column;
		gap: 1em;
	}

	.drag-drop {
		border: 1px dashed #ccc;
		padding: 2em;
		border-radius: 1em;
		display: inline-flex;
		align-content: center;
		cursor: pointer;
	}

	.drag-drop:hover {
		background-color: #3a6f3a;
	}

	.upload-preview {
		position: relative;
		width: 100%;
		min-height: 100px;
		display: flex;
		align-items: center;
		justify-content: center;
		overflow: hidden;
		border-radius: 0.5em;
		background: var(--card-bg, #2a2a2a);
	}

	.upload-preview img {
		max-width: 100%;
		max-height: 100px;
		display: block;
	}

	.upload-placeholder {
		width: 100%;
		height: 100px;
		background: linear-gradient(
			90deg,
			rgba(255, 255, 255, 0.05) 25%,
			rgba(255, 255, 255, 0.12) 50%,
			rgba(255, 255, 255, 0.05) 75%
		);
		background-size: 200% 100%;
		animation: upload-shimmer 1.2s ease-in-out infinite;
	}

	.upload-overlay {
		position: absolute;
		inset: 0;
		display: flex;
		align-items: center;
		justify-content: center;
		background: rgba(0, 0, 0, 0.55);
		color: #fff;
		font-size: 0.9em;
		font-weight: 600;
	}

	.upload-overlay--success {
		background: rgba(40, 120, 60, 0.75);
	}

	.uploaded-file p {
		margin: 0;
		padding: 0;
	}

	.upload-name {
		max-width: 180px;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
		text-align: center;
	}

	.uploaded-file {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 0.5em;
		border: 1px solid #666;
		border-radius: 0.5em;
		padding: 0.3em;
		width: 180px;
	}

	.uploaded-file--uploading {
		border-color: var(--accent-color, #5a9fd4);
	}

	.uploaded-file--success {
		border-color: #3a8f3a;
	}

	.uploaded-file--error {
		border-color: #c44;
	}

	.upload-progress {
		width: 100%;
		height: 0.5em;
	}

	.upload-status.error {
		color: #f88;
		font-size: 0.8em;
		text-align: center;
		word-break: break-word;
	}

	.image-preview {
		display: flex;
		flex-wrap: wrap;
		gap: 1em;
	}

	@keyframes upload-shimmer {
		0% {
			background-position: 200% 0;
		}
		100% {
			background-position: -200% 0;
		}
	}
</style>
