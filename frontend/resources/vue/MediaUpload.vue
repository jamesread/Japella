<template>
    <section>
        <h2>Media upload</h2>

        <p>Here you can upload images and videos to your media library.</p>

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
    import { onMounted } from 'vue';
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
            formData.append('media-file', file);

            window.client.uploadMedia(formData)
                .then((response) => {
                    if (response.standardResponse.success) {
                        alert(`File ${file.name} uploaded successfully.`);
                    } else {
                        alert(`Error uploading file ${file.name}: ${response.standardResponse.errorMessage}`);
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

  const curFiles = input.files;
  if (curFiles.length === 0) {
    const para = document.createElement("p");
    para.textContent = "No files currently selected for upload";
  } else {
    const list = document.createElement("ol");

    for (const file of curFiles) {
      const listItem = document.createElement("li");
      const para = document.createElement("p");
      if (validFileType(file)) {
        para.textContent = `File name ${file.name}, file size ${returnFileSize(
          file.size,
        )}.`;

        file.objectUrl = URL.createObjectURL(file);
        uploadedfiles.value.push(file);
      }

      list.appendChild(listItem);
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
