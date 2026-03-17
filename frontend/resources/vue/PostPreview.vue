<template>
	<div class="post-preview">
		<div class="post-header">
			<div class="social-account-info">
				<router-link
					v-if="post.socialAccountId"
					:to="{ name: 'socialAccountDetails', params: { id: post.socialAccountId } }"
					class="social-account-link"
				>
					<Icon :icon="post.socialAccountIcon || 'mdi:account'" />
					<span class="account-name">{{ post.authorName || post.socialAccountIdentity }}</span>
				</router-link>
				<div v-else class="social-account-link canned-post-label">
					<Icon icon="jam:box" />
					<span class="account-name">Canned Post</span>
				</div>
			</div>
			<button
				v-if="post.id || post.remoteId"
				@click="openDiagnosticDialog"
				class="diagnostic-button neutral small"
				title="Diagnostic Info"
			>
				<Icon icon="mdi:information-outline" />
			</button>
		</div>


		<div class="post-content-section">
			<div class="post-message" v-html="post.content"></div>
		</div>

		<div v-if="post.previewUrl || post.previewImageUrl" class="preview-card">
			<a
				v-if="post.previewUrl"
				:href="post.previewUrl"
				target="_blank"
				rel="noopener noreferrer"
				class="preview-card-link"
			>
				<div v-if="post.previewImageUrl" class="preview-image">
					<img :src="post.previewImageUrl" :alt="post.previewTitle || 'Preview image'" />
				</div>
				<div class="preview-content">
					<div v-if="post.previewTitle" class="preview-title">{{ post.previewTitle }}</div>
					<div v-if="post.previewDescription" class="preview-description">{{ post.previewDescription }}</div>
					<div v-if="post.previewUrl" class="preview-url">{{ getDomainFromUrl(post.previewUrl) }}</div>
				</div>
			</a>
			<div v-else class="preview-card-static">
				<div v-if="post.previewImageUrl" class="preview-image">
					<img :src="post.previewImageUrl" :alt="post.previewTitle || 'Preview image'" />
				</div>
				<div class="preview-content">
					<div v-if="post.previewTitle" class="preview-title">{{ post.previewTitle }}</div>
					<div v-if="post.previewDescription" class="preview-description">{{ post.previewDescription }}</div>
				</div>
			</div>
		</div>

		<div v-if="post.remoteUrl" class="post-actions">
			<a :href="post.remoteUrl" target="_blank" rel="noopener noreferrer" class="original-post-link">
				<Icon icon="mdi:open-in-new" width="16" height="16" />
				<span>View Original Post</span>
			</a>
		</div>

		<div v-if="post.postedDate || post.created" class="post-datetime">
			{{ formatDateTime(post.postedDate || post.created) }}
		</div>

	</div>

	<!-- Diagnostic Info Dialog -->
	<div v-if="showDiagnosticDialog" class="modal-overlay" @click.self="closeDiagnosticDialog">
		<div class="modal">
			<h3>Diagnostic Info</h3>
			<dl class="diagnostic-content">
				<dt v-if="post.id">ID</dt>
				<dd v-if="post.id" class="id-value">{{ post.id }}</dd>
				<dt v-if="post.remoteId">Remote ID</dt>
				<dd v-if="post.remoteId" class="id-value remote-id">{{ post.remoteId }}</dd>
			</dl>
			<div class="dialog-actions">
				<button class="neutral" @click="closeDiagnosticDialog">Close</button>
			</div>
		</div>
	</div>
</template>

<script setup>
	import { ref } from 'vue';
	import { Icon } from '@iconify/vue';

	// Props
	const props = defineProps({
		post: {
			type: Object,
			required: true
		}
	});

	// Diagnostic dialog state
	const showDiagnosticDialog = ref(false);

	function openDiagnosticDialog() {
		showDiagnosticDialog.value = true;
	}

	function closeDiagnosticDialog() {
		showDiagnosticDialog.value = false;
	}

	function formatDateTime(dateString) {
		if (!dateString) {
			return '';
		}

		try {
			// Parse the date string (format: "2006-01-02 15:04:05")
			// Try replacing space with T for ISO format, or parse directly
			let date = new Date(dateString.replace(' ', 'T'));
			if (isNaN(date.getTime())) {
				date = new Date(dateString);
			}
			if (isNaN(date.getTime())) {
				return dateString; // Return original if can't parse
			}

			// Format as: "Jan 15, 2026 4:54pm"
			const months = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec'];
			const month = months[date.getMonth()];
			const day = date.getDate();
			const year = date.getFullYear();
			const hours = date.getHours();
			const minutes = date.getMinutes();
			const ampm = hours >= 12 ? 'pm' : 'am';
			const displayHours = hours % 12 || 12;
			const timeStr = minutes > 0
				? `${displayHours}:${minutes.toString().padStart(2, '0')}${ampm}`
				: `${displayHours}${ampm}`;

			return `${month} ${day}, ${year} ${timeStr}`;
		} catch (e) {
			return dateString;
		}
	}

	function getDomainFromUrl(url) {
		if (!url) return '';
		try {
			const urlObj = new URL(url);
			return urlObj.hostname.replace('www.', '');
		} catch (e) {
			return url;
		}
	}

	function statusClass(post) {
		if (post.state === 'error') return 'bad';
		if (post.state === 'pending' || post.state === 'scheduled') return 'note';
		if (post.state === 'completed') return 'good';
		return '';
	}

	function statusText(post) {
		if (post.state === 'error') return 'Error';
		if (post.state === 'pending' || post.state === 'scheduled') return 'Scheduled';
		if (post.state === 'completed') return 'Completed';
		return 'Unknown';
	}
</script>

<style scoped>
.post-preview {
	max-width: 800px;
	margin: 0 auto;
	padding: 1rem;
	border-radius: 0.5rem;
    line-height: 2;
	position: relative;
}

.post-header {
	display: flex;
	align-items: start;
	justify-content: space-between;
	gap: 0.5rem;
	margin-bottom: 0.5rem;
}

.social-account-info {
	display: flex;
	align-items: center;
	gap: 0.5rem;
	font-size: 1.1rem;
	font-weight: 500;
}

.diagnostic-button {
	display: flex;
	align-items: center;
	justify-content: center;
}

.diagnostic-content {
	margin-bottom: 1rem;
}

.diagnostic-content dt {
	font-weight: 500;
	font-size: 0.875rem;
	color: var(--text-secondary, #666);
	margin-top: 0.75rem;
	margin-bottom: 0.25rem;
}

.diagnostic-content dt:first-child {
	margin-top: 0;
}

.diagnostic-content dd {
	margin: 0 0 0.75rem 0;
	font-size: 0.875rem;
}

.diagnostic-content dd.remote-id {
	font-size: 0.75rem;
}

.id-value {
	font-family: monospace;
	background-color: var(--bg-secondary, #f5f5f5);
	padding: 0.125rem 0.375rem;
	border-radius: 0.25rem;
	word-break: break-all;
	max-width: 400px;
	overflow-wrap: break-word;
	display: inline-block;
}

.social-account-link {
	display: flex;
	align-items: center;
	gap: 0.5rem;
	text-decoration: none;
	color: var(--link-color, #007bff);
}

.social-account-link:hover {
	text-decoration: underline;
}

.canned-post-label {
	cursor: default;
}

.canned-post-label:hover {
	text-decoration: none;
}

.account-name {
    font-weight: bold;
}

.post-message {
	font-weight: 500;
	word-wrap: break-word;
}

.post-message :deep(a) {
	color: var(--link-color, #007bff);
	text-decoration: underline;
}

.post-message :deep(a:hover) {
	color: var(--link-hover-color, #0056b3);
}

.post-message :deep(p) {
	margin: 0.5em 0;
}

.post-message :deep(p:first-child) {
	margin-top: 0;
}

.post-message :deep(p:last-child) {
	margin-bottom: 0;
}

.post-content-section {
	margin-bottom: 2rem;
}

.post-content-section h3 {
	margin: 0 0 1rem 0;
	color: var(--text-primary, #333);
	font-size: 1.2rem;
}

.post-actions {
	margin-top: 1rem;
	padding-top: 1rem;
	border-top: 1px solid var(--border-color, #4e5965);
}

.original-post-link {
	display: flex;
	align-items: center;
	gap: 0.5rem;
	color: var(--link-color, #007bff);
	text-decoration: none;
	font-size: 0.9rem;
	transition: color 0.2s ease;
}

.original-post-link:hover {
	color: var(--link-hover-color, #0056b3);
	text-decoration: underline;
}

/* Modal styles */
.modal-overlay {
	position: fixed;
	top: 0;
	left: 0;
	width: 100%;
	height: 100%;
	background-color: rgba(0, 0, 0, 0.5);
	display: flex;
	justify-content: center;
	align-items: center;
	z-index: 1000;
}

.modal {
	background: white;
	padding: 2rem;
	border-radius: 0.5rem;
	box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1);
	min-width: 400px;
	max-width: 500px;
}

.modal h3 {
	margin: 0 0 1rem 0;
	color: #333;
}

.dialog-actions {
	display: flex;
	gap: 0.5rem;
	justify-content: flex-end;
	margin-top: 1.5rem;
}

.dialog-actions button {
	padding: 0.5rem 1rem;
	border-radius: 0.25rem;
	border: none;
	cursor: pointer;
	font-size: 0.9rem;
	display: flex;
	align-items: center;
	gap: 0.5rem;
	transition: all 0.2s ease;
}

.dialog-actions button.neutral {
	background-color: #f0f0f0;
	color: #333;
}

.dialog-actions button.neutral:hover {
	background-color: #e0e0e0;
}

.post-datetime {
	position: absolute;
	bottom: 1rem;
	right: 1rem;
	font-size: 0.875rem;
	color: var(--text-secondary, #666);
	font-style: italic;
}

.preview-card {
	margin: 1rem 0;
	border: 1px solid var(--border-color, #e5e7eb);
	border-radius: 0.5rem;
	overflow: hidden;
	background-color: var(--bg-primary, #ffffff);
}

.preview-card-link {
	display: flex;
	text-decoration: none;
	color: inherit;
	transition: opacity 0.2s ease;
}

.preview-card-link:hover {
	opacity: 0.8;
}

.preview-card-static {
	display: flex;
}

.preview-image {
	flex-shrink: 0;
	width: 200px;
	height: 150px;
	overflow: hidden;
	background-color: var(--bg-secondary, #f5f5f5);
}

.preview-image img {
	width: 100%;
	height: 100%;
	object-fit: cover;
}

.preview-content {
	flex: 1;
	padding: 1rem;
	display: flex;
	flex-direction: column;
	gap: 0.5rem;
}

.preview-title {
	font-weight: 600;
	font-size: 1rem;
	color: var(--text-primary, #333);
	line-height: 1.4;
}

.preview-description {
	font-size: 0.875rem;
	color: var(--text-secondary, #666);
	line-height: 1.5;
	display: -webkit-box;
	-webkit-line-clamp: 2;
	-webkit-box-orient: vertical;
	overflow: hidden;
}

.preview-url {
	font-size: 0.75rem;
	color: var(--text-secondary, #999);
	text-transform: uppercase;
	letter-spacing: 0.5px;
	margin-top: auto;
}

@media (max-width: 600px) {
	.preview-image {
		width: 120px;
		height: 90px;
	}

	.preview-content {
		padding: 0.75rem;
	}
}

</style>
