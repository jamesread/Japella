<template>
	<div v-if="showPrompt" class="pwa-install-prompt">
		<div class="pwa-install-content">
			<Icon icon="mdi:download" width="24" height="24" />
			<div class="pwa-install-text">
				<strong>{{ $t('pwa.install.title', 'Install Japella') }}</strong>
				<span>{{ $t('pwa.install.message', 'Install this app on your device for a better experience') }}</span>
			</div>
			<div class="pwa-install-actions">
				<button @click="handleInstall" class="pwa-install-button">
					{{ $t('pwa.install.button', 'Install') }}
				</button>
				<button @click="dismissPrompt" class="pwa-dismiss-button" aria-label="Dismiss">
					<Icon icon="mdi:close" width="20" height="20" />
				</button>
			</div>
		</div>
	</div>
</template>

<script setup>
import { ref, onMounted, onBeforeUnmount } from 'vue';
import { Icon } from '@iconify/vue';
import { useI18n } from 'vue-i18n';

const { t } = useI18n();

const showPrompt = ref(false);
let deferredPrompt = null;
const dismissedKey = 'pwa-install-dismissed';

onMounted(() => {
	// Check if already installed
	if (window.matchMedia('(display-mode: standalone)').matches || 
	    window.navigator.standalone === true) {
		return; // Already installed
	}

	// Check if user previously dismissed
	const dismissed = localStorage.getItem(dismissedKey);
	if (dismissed) {
		const dismissedTime = parseInt(dismissed, 10);
		const daysSinceDismissed = (Date.now() - dismissedTime) / (1000 * 60 * 60 * 24);
		// Show again after 7 days
		if (daysSinceDismissed < 7) {
			return;
		}
	}

	// Listen for the beforeinstallprompt event
	window.addEventListener('beforeinstallprompt', handleBeforeInstallPrompt);
	
	// Check if app is already installed
	window.addEventListener('appinstalled', handleAppInstalled);
});

onBeforeUnmount(() => {
	window.removeEventListener('beforeinstallprompt', handleBeforeInstallPrompt);
	window.removeEventListener('appinstalled', handleAppInstalled);
});

function handleBeforeInstallPrompt(e) {
	// Prevent the default browser install prompt
	e.preventDefault();
	
	// Store the event for later use
	deferredPrompt = e;
	
	// Show our custom prompt
	showPrompt.value = true;
}

function handleAppInstalled() {
	// Hide the prompt if app gets installed
	showPrompt.value = false;
	deferredPrompt = null;
}

async function handleInstall() {
	if (!deferredPrompt) {
		return;
	}

	// Show the install prompt
	deferredPrompt.prompt();

	// Wait for the user to respond
	const { outcome } = await deferredPrompt.userChoice;

	if (outcome === 'accepted') {
		console.log('User accepted the install prompt');
	} else {
		console.log('User dismissed the install prompt');
	}

	// Clear the deferred prompt
	deferredPrompt = null;
	showPrompt.value = false;
}

function dismissPrompt() {
	showPrompt.value = false;
	// Store dismissal timestamp
	localStorage.setItem(dismissedKey, Date.now().toString());
}
</script>

<style scoped>
.pwa-install-prompt {
	position: fixed;
	bottom: 20px;
	left: 50%;
	transform: translateX(-50%);
	background-color: #242424;
	border: 1px solid #4CAF50;
	border-radius: 8px;
	padding: 16px;
	box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);
	z-index: 1000;
	max-width: 90%;
	width: 500px;
	animation: slideUp 0.3s ease-out;
}

@keyframes slideUp {
	from {
		transform: translateX(-50%) translateY(100px);
		opacity: 0;
	}
	to {
		transform: translateX(-50%) translateY(0);
		opacity: 1;
	}
}

.pwa-install-content {
	display: flex;
	align-items: center;
	gap: 12px;
}

.pwa-install-text {
	flex: 1;
	display: flex;
	flex-direction: column;
	gap: 4px;
}

.pwa-install-text strong {
	color: #4CAF50;
	font-size: 16px;
}

.pwa-install-text span {
	color: #e0e0e0;
	font-size: 14px;
}

.pwa-install-actions {
	display: flex;
	align-items: center;
	gap: 8px;
}

.pwa-install-button {
	background-color: #4CAF50;
	color: white;
	border: none;
	border-radius: 4px;
	padding: 8px 16px;
	font-size: 14px;
	font-weight: 500;
	cursor: pointer;
	transition: background-color 0.2s ease;
}

.pwa-install-button:hover {
	background-color: #45a049;
}

.pwa-install-button:active {
	background-color: #3d8b40;
}

.pwa-dismiss-button {
	background: transparent;
	border: none;
	color: #e0e0e0;
	cursor: pointer;
	padding: 4px;
	display: flex;
	align-items: center;
	justify-content: center;
	border-radius: 4px;
	transition: background-color 0.2s ease, color 0.2s ease;
}

.pwa-dismiss-button:hover {
	background-color: rgba(255, 255, 255, 0.1);
	color: white;
}

@media (max-width: 600px) {
	.pwa-install-prompt {
		width: calc(100% - 40px);
		max-width: none;
		left: 20px;
		right: 20px;
		transform: none;
	}

	.pwa-install-content {
		flex-direction: column;
		align-items: flex-start;
	}

	.pwa-install-actions {
		width: 100%;
		justify-content: flex-end;
	}
}
</style>
