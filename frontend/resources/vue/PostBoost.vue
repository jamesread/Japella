<template>
	<div class="post-boost">
		<div class="boost-header">
			<div class="boost-info">
				<Icon icon="mdi:repeat" class="boost-icon" />
				<router-link 
					v-if="post.socialAccountId" 
					:to="{ name: 'socialAccountDetails', params: { id: post.socialAccountId } }" 
					class="social-account-link"
				>
					<Icon :icon="post.socialAccountIcon || 'mdi:account'" />
					<span class="account-name">{{ boostActor || post.authorName || post.socialAccountIdentity }}</span>
				</router-link>
				<div v-else class="social-account-link">
					<Icon icon="mdi:account" />
					<span class="account-name">{{ boostActor || 'Unknown' }}</span>
				</div>
				<span class="boost-text">boosted</span>
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

		<div class="boosted-content">
			<div v-if="boostedObjectUrl" class="boosted-object-link">
				<a :href="boostedObjectUrl" target="_blank" rel="noopener noreferrer" class="boosted-link">
					<Icon icon="mdi:open-in-new" width="16" height="16" />
					<span>View Boosted Post</span>
				</a>
			</div>
			<div v-else class="boosted-object-info">
				<span class="boosted-object-label">Boosted Object:</span>
				<span class="boosted-object-value">{{ boostedObjectUrl || 'Unknown' }}</span>
			</div>
		</div>

		<div v-if="post.remoteUrl" class="post-actions">
			<a :href="post.remoteUrl" target="_blank" rel="noopener noreferrer" class="original-post-link">
				<Icon icon="mdi:open-in-new" width="16" height="16" />
				<span>View Boost Activity</span>
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
	import { ref, computed } from 'vue';
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

	// Parse ActivityStreams JSON from remoteId
	const activityStreams = computed(() => {
		if (!props.post.remoteId) {
			return null;
		}

		try {
			// Try to parse as JSON
			const parsed = JSON.parse(props.post.remoteId);
			if (parsed && parsed['@context'] && parsed.type) {
				return parsed;
			}
		} catch (e) {
			// Not JSON, return null
			return null;
		}

		return null;
	});

	// Extract actor from remoteUrl if remoteId doesn't contain JSON
	const actorFromUrl = computed(() => {
		if (activityStreams.value) {
			return null; // Already extracted from JSON
		}

		if (!props.post.remoteUrl) {
			return null;
		}

		// Extract username from URL like "https://fosstodon.org/users/jimsalter/statuses/..."
		const match = props.post.remoteUrl.match(/\/users\/([^\/]+)/);
		if (match) {
			return match[1];
		}

		// Try ActivityPub actor format
		const actorMatch = props.post.remoteUrl.match(/@([^\/@]+)@([^\/]+)/);
		if (actorMatch) {
			return actorMatch[1];
		}

		return null;
	});

	// Check if this is a boost (Announce type)
	const isBoost = computed(() => {
		return activityStreams.value && activityStreams.value.type === 'Announce';
	});

	// Extract boost actor (who boosted)
	const boostActor = computed(() => {
		// First try to get from ActivityStreams JSON
		if (activityStreams.value) {
			const actor = activityStreams.value.actor;
			if (typeof actor === 'string') {
				// Extract username from URL like "https://fosstodon.org/users/jimsalter"
				const match = actor.match(/\/users\/([^\/]+)/);
				if (match) {
					return match[1];
				}
				return actor;
			}
		}

		// Fall back to extracting from remoteUrl
		if (actorFromUrl.value) {
			return actorFromUrl.value;
		}

		// Fall back to author name from post
		return props.post.authorName || props.post.socialAccountIdentity;
	});

	// Extract boosted object URL
	const boostedObjectUrl = computed(() => {
		// First try to get from ActivityStreams JSON
		if (activityStreams.value) {
			const obj = activityStreams.value.object;
			if (typeof obj === 'string') {
				return obj;
			}
		}

		// If remoteUrl is an activity URL, try to extract the object URL
		// Activity URLs often look like: .../statuses/123/activity
		// The object would be: .../statuses/123
		if (props.post.remoteUrl && props.post.remoteUrl.endsWith('/activity')) {
			return props.post.remoteUrl.replace('/activity', '');
		}

		return null;
	});

	// Display remote ID (shortened if it's a URL)
	const displayRemoteId = computed(() => {
		if (!props.post.remoteId) {
			return '';
		}

		// If it's JSON, show a shortened version
		if (activityStreams.value) {
			return `${activityStreams.value.type} activity`;
		}

		// If it's a URL, show just the last part
		if (props.post.remoteId.startsWith('http')) {
			const parts = props.post.remoteId.split('/');
			return parts[parts.length - 1] || props.post.remoteId;
		}

		return props.post.remoteId;
	});
</script>

<style scoped>
.post-boost {
	max-width: 800px;
	margin: 0 auto;
	padding: 1rem;
	border-radius: 0.5rem;
	border-left: 3px solid var(--boost-color, #6366f1);
	background-color: var(--bg-secondary, #f9fafb);
	position: relative;
}

.boost-header {
	display: flex;
	align-items: start;
	justify-content: space-between;
	gap: 0.5rem;
	margin-bottom: 1rem;
}

.boost-info {
	display: flex;
	align-items: center;
	gap: 0.5rem;
	font-size: 1rem;
	font-weight: 500;
}

.boost-icon {
	color: var(--boost-color, #6366f1);
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

.account-name {
	font-weight: bold;
}

.boost-text {
	color: var(--text-secondary, #666);
	font-weight: normal;
	font-style: italic;
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

.boosted-content {
	margin: 1rem 0;
	padding: 1rem;
	background-color: var(--bg-primary, #ffffff);
	border-radius: 0.25rem;
	border: 1px solid var(--border-color, #e5e7eb);
}

.boosted-object-link {
	margin-bottom: 0.5rem;
}

.boosted-link {
	display: inline-flex;
	align-items: center;
	gap: 0.5rem;
	color: var(--link-color, #007bff);
	text-decoration: none;
	font-size: 0.9rem;
	transition: color 0.2s ease;
}

.boosted-link:hover {
	color: var(--link-hover-color, #0056b3);
	text-decoration: underline;
}

.boosted-object-info {
	display: flex;
	flex-direction: column;
	gap: 0.25rem;
}

.boosted-object-label {
	font-size: 0.875rem;
	color: var(--text-secondary, #666);
	font-weight: 500;
}

.boosted-object-value {
	font-family: monospace;
	font-size: 0.875rem;
	color: var(--text-primary, #333);
	word-break: break-all;
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
</style>
