<template>
    <section>
        <h2>Media upload</h2>

        <p>Here you can upload images and videos to your media library. You can also paste images from your clipboard (Ctrl+V).</p>

        <form id="media-upload-form" @submit.prevent="submitForm">
            <fieldset style = "flex-direction: column; gap: 1em;">
                <div ref = "previewEl" class = "image-preview">
                    <div class = "uploaded-file" v-for="(file, index) in uploadedfiles" :key="index">
                        <img :src="file.objectUrl" />

                        <p><small>{{ returnFileSize(file.size) }}</small></p>
                    </div>
                </div>
                <label for="media-file" class = "drag-drop">{{ t('section.media.upload.label') }}:
                <input type="file" multiple id="media-file" name="media-file" accept="image/*,video/*" ref = "fileupload" />
                </label>
            </fieldset>
            <fieldset>
                <button id="submit" type="submit">{{ t('section.media.upload.submit') }}</button>
            </fieldset>
        </form>
    </section>

</template>

<script setup>
    import { ref } from 'vue';
    import { useI18n } from 'vue-i18n';
    import { onMounted, onUnmounted } from 'vue';
    const { t } = useI18n()

    const fileupload = ref(null);
    const previewEl = ref(null);
    const uploadedfiles = ref([]);

    function submitForm() {
        const input = fileupload.value;
        const curFiles = input.files;

        if (curFiles.length === 0) {
            alert("No files currently selected for upload");
            return;
        }

        for (const file of curFiles) {
            if (!validFileType(file)) {
                alert(`File type ${file.type} is not supported.`);
                continue;
            }

            const formData = new FormData();
            formData.append('file', file);

            fetch('/upload', {
                method: 'POST',
                body: formData,
                credentials: 'include',
            })
                .then(async (response) => {
                    const text = await response.text();
                    if (response.ok) {
                        alert(`File ${file.name} uploaded successfully.`);
                        window.dispatchEvent(new CustomEvent('media-uploaded'));
                    } else {
                        alert(`Error uploading file ${file.name}: ${text || response.statusText}`);
                    }
                })
                .catch((error) => {
                    console.error('Upload failed:', error);
                    alert(`Error uploading file ${file.name}: ${error.message}`);
                });
        }
    }

    onMounted(() => {
        const fileInput = fileupload.value;
        if (!fileInput) return;
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
            }
        });

        fileInput.addEventListener('change', updateImageDisplay);

        const handlePaste = (event) => {
            const items = event.clipboardData?.items;
            if (!items) return;

            const pastedImages = [];
            for (const item of items) {
                if (item.type.startsWith('image/')) {
                    const file = item.getAsFile();
                    if (file) pastedImages.push(file);
                }
            }
            if (pastedImages.length === 0) return;

            event.preventDefault();
            const dt = new DataTransfer();
            for (const f of fileInput.files) dt.items.add(f);
            for (const f of pastedImages) dt.items.add(f);
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
    });

const fileTypes = [
  "image/apng",
  "image/bmp",
  "image/gif",
  "image/jpeg",
  "image/pjpeg",
  "image/png",
  "image/svg+xml",
  "image/tiff",
  "image/webp",
  "image/x-icon",
];

function validFileType(file) {
  return fileTypes.includes(file.type);
}

function returnFileSize(number) {
  if (number < 1e3) {
    return `${number} bytes`;
  } else if (number >= 1e3 && number < 1e6) {
    return `${(number / 1e3).toFixed(1)} KB`;
  }
  return `${(number / 1e6).toFixed(1)} MB`;
}

function updateImageDisplay() {
  const input = fileupload.value;
  if (!input) return;

  for (const f of uploadedfiles.value) {
    if (f.objectUrl) URL.revokeObjectURL(f.objectUrl);
  }
  uploadedfiles.value = [];

  const curFiles = input.files;
  for (const file of curFiles) {
    if (validFileType(file)) {
      file.objectUrl = URL.createObjectURL(file);
      uploadedfiles.value.push(file);
    }
  }
}

</script>

<style scoped>
    label {
        border: 1px dashed #ccc;
        padding: 2em;
        border-radius: 1em;
        display: inline-flex;
        align-content: center;
    }

    label:hover {
        background-color: #3a6f3a;
    }

    .uploaded-file img {
        max-width: 600px;
        max-height: 100px;
        border-radius: 0.5em;
    }

    .uploaded-file p {
        margin: 0;
        padding: 0;
    }

    .uploaded-file {
        display: flex;
        flex-direction: column;
        align-items: center;
        gap: 0.5em;
        border: 1px solid #666;
        border-radius: 0.5em;
        padding: .3em;
    }

    .image-preview {
        display: flex;
        flex-wrap: wrap;
        gap: 1em;
    }
</style>
