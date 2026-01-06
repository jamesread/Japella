<template>
	<Section
		title="Browser Diagnostics"
		subtitle="View browser and system information for troubleshooting"
	>
		<template #toolbar>
			<button @click="refreshDiagnostics" class="neutral">
				<Icon icon="material-symbols:refresh" />
			</button>
		</template>

		<div v-if="loading" class="icon-and-text">
			<Icon icon="eos-icons:loading" width="24" height="24" />
			<span>Loading diagnostics...</span>
		</div>

		<div v-else>
			<!-- Browser Information -->
			<h3>Browser Information</h3>
			<dl class="diagnostics-info">
				<dt>User Agent</dt>
				<dd>{{ diagnostics.userAgent }}</dd>

				<dt>Browser Name</dt>
				<dd>{{ diagnostics.browserName }}</dd>

				<dt>Browser Version</dt>
				<dd>{{ diagnostics.browserVersion }}</dd>

				<dt>Platform</dt>
				<dd>{{ diagnostics.platform }}</dd>

				<dt>Language</dt>
				<dd>{{ diagnostics.language }}</dd>

				<dt>Languages</dt>
				<dd>{{ diagnostics.languages.join(', ') }}</dd>

				<dt>Screen Resolution</dt>
				<dd>{{ diagnostics.screenWidth }} × {{ diagnostics.screenHeight }}</dd>

				<dt>Window Size</dt>
				<dd>{{ diagnostics.windowWidth }} × {{ diagnostics.windowHeight }}</dd>

				<dt>Color Depth</dt>
				<dd>{{ diagnostics.colorDepth }} bits</dd>

				<dt>Pixel Ratio</dt>
				<dd>{{ diagnostics.pixelRatio }}</dd>
			</dl>

			<!-- System Information -->
			<h3>System Information</h3>
			<dl class="diagnostics-info">
				<dt>Timezone</dt>
				<dd>{{ diagnostics.timezone }}</dd>

				<dt>Timezone Offset</dt>
				<dd>{{ diagnostics.timezoneOffset }} minutes</dd>

				<dt>Cookies Enabled</dt>
				<dd>{{ diagnostics.cookiesEnabled ? 'Yes' : 'No' }}</dd>

				<dt>Local Storage</dt>
				<dd>{{ diagnostics.localStorageAvailable ? 'Available' : 'Not Available' }}</dd>

				<dt>Session Storage</dt>
				<dd>{{ diagnostics.sessionStorageAvailable ? 'Available' : 'Not Available' }}</dd>

				<dt>IndexedDB</dt>
				<dd>{{ diagnostics.indexedDBAvailable ? 'Available' : 'Not Available' }}</dd>

				<dt>Secure Context</dt>
				<dd>{{ diagnostics.isSecureContext ? 'Yes' : 'No' }}</dd>

				<dt>Online Status</dt>
				<dd>{{ diagnostics.online ? 'Online' : 'Offline' }}</dd>
			</dl>

			<!-- Feature Detection -->
			<h3>Feature Detection</h3>
			<dl class="diagnostics-info">
				<dt>WebSocket</dt>
				<dd>{{ diagnostics.webSocketSupported ? 'Supported' : 'Not Supported' }}</dd>

				<dt>Web Workers</dt>
				<dd>{{ diagnostics.webWorkersSupported ? 'Supported' : 'Not Supported' }}</dd>

				<dt>Service Workers</dt>
				<dd>{{ diagnostics.serviceWorkersSupported ? 'Supported' : 'Not Supported' }}</dd>

				<dt>Fetch API</dt>
				<dd>{{ diagnostics.fetchSupported ? 'Supported' : 'Not Supported' }}</dd>

				<dt>Geolocation</dt>
				<dd>{{ diagnostics.geolocationSupported ? 'Supported' : 'Not Supported' }}</dd>

				<dt>Notifications</dt>
				<dd>{{ diagnostics.notificationsSupported ? 'Supported' : 'Not Supported' }}</dd>
			</dl>

			<!-- Performance Information -->
			<h3>Performance Information</h3>
			<dl class="diagnostics-info">
				<dt>Hardware Concurrency</dt>
				<dd>{{ diagnostics.hardwareConcurrency || 'Unknown' }} cores</dd>

				<dt>Device Memory</dt>
				<dd>{{ diagnostics.deviceMemory ? diagnostics.deviceMemory + ' GB' : 'Unknown' }}</dd>

				<dt>Connection Type</dt>
				<dd>{{ diagnostics.connectionType || 'Unknown' }}</dd>

				<dt>Connection Effective Type</dt>
				<dd>{{ diagnostics.effectiveType || 'Unknown' }}</dd>
			</dl>

			<!-- PWA Information -->
			<h3>PWA Information</h3>
			<dl class="diagnostics-info">
				<dt>Service Worker Registered</dt>
				<dd>{{ diagnostics.serviceWorkerRegistered ? 'Yes' : 'No' }}</dd>

				<dt>Service Worker Version</dt>
				<dd>{{ diagnostics.serviceWorkerVersion || 'Not Available' }}</dd>

				<dt>Service Worker State</dt>
				<dd>{{ diagnostics.serviceWorkerState || 'Not Available' }}</dd>

				<dt>Service Worker Scope</dt>
				<dd>{{ diagnostics.serviceWorkerScope || 'Not Available' }}</dd>
			</dl>
		</div>
	</Section>
</template>

<script setup>
	import { ref, onMounted } from 'vue';
	import { Icon } from '@iconify/vue';
	import Section from 'picocrank/vue/components/Section.vue';

	const loading = ref(true);
	const diagnostics = ref({});

	function detectBrowser() {
		const ua = navigator.userAgent;
		let browserName = 'Unknown';
		let browserVersion = 'Unknown';

		if (ua.indexOf('Chrome') > -1 && ua.indexOf('Edg') === -1) {
			browserName = 'Chrome';
			const match = ua.match(/Chrome\/(\d+)/);
			browserVersion = match ? match[1] : 'Unknown';
		} else if (ua.indexOf('Firefox') > -1) {
			browserName = 'Firefox';
			const match = ua.match(/Firefox\/(\d+)/);
			browserVersion = match ? match[1] : 'Unknown';
		} else if (ua.indexOf('Safari') > -1 && ua.indexOf('Chrome') === -1) {
			browserName = 'Safari';
			const match = ua.match(/Version\/(\d+)/);
			browserVersion = match ? match[1] : 'Unknown';
		} else if (ua.indexOf('Edg') > -1) {
			browserName = 'Edge';
			const match = ua.match(/Edg\/(\d+)/);
			browserVersion = match ? match[1] : 'Unknown';
		} else if (ua.indexOf('Opera') > -1 || ua.indexOf('OPR') > -1) {
			browserName = 'Opera';
			const match = ua.match(/(?:Opera|OPR)\/(\d+)/);
			browserVersion = match ? match[1] : 'Unknown';
		}

		return { browserName, browserVersion };
	}

	function getConnectionInfo() {
		const connection = navigator.connection || navigator.mozConnection || navigator.webkitConnection;
		return {
			type: connection?.type || 'Unknown',
			effectiveType: connection?.effectiveType || 'Unknown'
		};
	}

	async function getServiceWorkerInfo() {
		if (!('serviceWorker' in navigator)) {
			return {
				registered: false,
				version: null,
				state: null,
				scope: null
			};
		}

		try {
			const registration = await navigator.serviceWorker.getRegistration();
			if (!registration) {
				return {
					registered: false,
					version: null,
					state: null,
					scope: null
				};
			}

			const worker = registration.active || registration.waiting || registration.installing;
			let version = null;

			// Try to get version from service worker
			if (worker) {
				try {
					const messageChannel = new MessageChannel();
					const versionPromise = new Promise((resolve) => {
						messageChannel.port1.onmessage = (event) => {
							if (event.data && event.data.version) {
								resolve(event.data.version);
							} else {
								resolve(null);
							}
						};
						// Timeout after 1 second
						setTimeout(() => resolve(null), 1000);
					});

					worker.postMessage({ type: 'GET_VERSION' }, [messageChannel.port2]);
					version = await versionPromise;
				} catch (error) {
					console.error('Error getting service worker version:', error);
				}
			}

			return {
				registered: true,
				version: version || 'Unknown',
				state: worker ? worker.state : 'Not Available',
				scope: registration.scope
			};
		} catch (error) {
			console.error('Error getting service worker info:', error);
			return {
				registered: false,
				version: null,
				state: null,
				scope: null
			};
		}
	}

	async function collectDiagnostics() {
		const { browserName, browserVersion } = detectBrowser();
		const connectionInfo = getConnectionInfo();
		const swInfo = await getServiceWorkerInfo();

		return {
			userAgent: navigator.userAgent,
			browserName,
			browserVersion,
			platform: navigator.platform,
			language: navigator.language,
			languages: navigator.languages || [navigator.language],
			screenWidth: screen.width,
			screenHeight: screen.height,
			windowWidth: window.innerWidth,
			windowHeight: window.innerHeight,
			colorDepth: screen.colorDepth,
			pixelRatio: window.devicePixelRatio || 1,
			timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
			timezoneOffset: new Date().getTimezoneOffset(),
			cookiesEnabled: navigator.cookieEnabled,
			localStorageAvailable: (() => {
				try {
					const test = '__localStorage_test__';
					localStorage.setItem(test, test);
					localStorage.removeItem(test);
					return true;
				} catch {
					return false;
				}
			})(),
			sessionStorageAvailable: (() => {
				try {
					const test = '__sessionStorage_test__';
					sessionStorage.setItem(test, test);
					sessionStorage.removeItem(test);
					return true;
				} catch {
					return false;
				}
			})(),
			indexedDBAvailable: !!window.indexedDB,
			isSecureContext: window.isSecureContext || false,
			online: navigator.onLine,
			webSocketSupported: typeof WebSocket !== 'undefined',
			webWorkersSupported: typeof Worker !== 'undefined',
			serviceWorkersSupported: 'serviceWorker' in navigator,
			fetchSupported: typeof fetch !== 'undefined',
			geolocationSupported: 'geolocation' in navigator,
			notificationsSupported: 'Notification' in window,
			hardwareConcurrency: navigator.hardwareConcurrency,
			deviceMemory: navigator.deviceMemory,
			connectionType: connectionInfo.type,
			effectiveType: connectionInfo.effectiveType,
			serviceWorkerRegistered: swInfo.registered,
			serviceWorkerVersion: swInfo.version,
			serviceWorkerState: swInfo.state,
			serviceWorkerScope: swInfo.scope
		};
	}

	async function refreshDiagnostics() {
		loading.value = true;
		try {
			diagnostics.value = await collectDiagnostics();
		} catch (error) {
			console.error('Error collecting diagnostics:', error);
		} finally {
			loading.value = false;
		}
	}

	onMounted(() => {
		refreshDiagnostics();
	});
</script>

<style scoped>
	.diagnostics-info {
		display: grid;
		grid-template-columns: 200px 1fr;
		column-gap: 1em;
		row-gap: 0.5em;
		margin: 0 0 2em 0;
	}

	.diagnostics-info dt {
		font-weight: bold;
	}

	.diagnostics-info dd {
		margin: 0;
		word-break: break-word;
	}

	h3 {
		margin-top: 2em;
		margin-bottom: 1em;
	}

	h3:first-child {
		margin-top: 0;
	}
</style>
