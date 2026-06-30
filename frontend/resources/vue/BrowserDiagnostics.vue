<template>
	<Section
		title="Browser Diagnostics"
		subtitle="Browser and client environment information for troubleshooting"
		classes="browser-diagnostics"
	>
		<template #toolbar>
			<button type="button" class="neutral" title="Refresh" :disabled="loading" @click="refreshDiagnostics">
				<Icon icon="material-symbols:refresh" />
			</button>
		</template>

		<p v-if="errorMessage" class="inline-notification error">{{ errorMessage }}</p>

		<p v-if="loading" class="muted">Loading diagnostics…</p>

		<ReadOnlyTextArea
			ref="diagnosticsRef"
			v-model="diagnosticsYaml"
			label="Diagnostics (YAML)"
			:rows="22"
			markdown-ticks
			markdown-lang="yaml"
		/>
	</Section>
</template>

<script setup>
	import { ref, onMounted, nextTick } from 'vue';
	import { Icon } from '@iconify/vue';
	import Section from 'picocrank/vue/components/Section.vue';
	import ReadOnlyTextArea from 'picocrank/vue/components/ReadOnlyTextArea.vue';

	const loading = ref(true);
	const errorMessage = ref('');
	const diagnostics = ref({});
	const diagnosticsYaml = ref('');
	const diagnosticsRef = ref(null);

	function yamlValue(value) {
		if (value === null || value === undefined || value === '') {
			return 'unknown';
		}
		if (typeof value === 'boolean') {
			return value ? 'true' : 'false';
		}
		return String(value);
	}

	function yamlList(items) {
		if (!Array.isArray(items) || items.length === 0) {
			return 'none';
		}
		return items.join(', ');
	}

	function yamlAvailable(value) {
		return value ? 'available' : 'not available';
	}

	function yamlSupported(value) {
		return value ? 'supported' : 'not supported';
	}

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
			effectiveType: connection?.effectiveType || 'Unknown',
		};
	}

	async function getServiceWorkerInfo() {
		if (!('serviceWorker' in navigator)) {
			return {
				registered: false,
				version: null,
				state: null,
				scope: null,
			};
		}

		try {
			const registration = await navigator.serviceWorker.getRegistration();
			if (!registration) {
				return {
					registered: false,
					version: null,
					state: null,
					scope: null,
				};
			}

			const worker = registration.active || registration.waiting || registration.installing;
			let version = null;

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
				scope: registration.scope,
			};
		} catch (error) {
			console.error('Error getting service worker info:', error);
			return {
				registered: false,
				version: null,
				state: null,
				scope: null,
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
			serviceWorkerScope: swInfo.scope,
		};
	}

	async function buildDiagnosticsYaml() {
		diagnosticsYaml.value = '';
		await nextTick();

		const area = diagnosticsRef.value;
		if (!area) {
			return;
		}

		const d = diagnostics.value;

		area.appendSection('Browser');
		area.appendYamlProperty('userAgent', yamlValue(d.userAgent));
		area.appendYamlProperty('browserName', yamlValue(d.browserName));
		area.appendYamlProperty('browserVersion', yamlValue(d.browserVersion));
		area.appendYamlProperty('platform', yamlValue(d.platform));
		area.appendYamlProperty('language', yamlValue(d.language));
		area.appendYamlProperty('languages', yamlList(d.languages));
		area.appendYamlProperty('screenWidth', yamlValue(d.screenWidth));
		area.appendYamlProperty('screenHeight', yamlValue(d.screenHeight));
		area.appendYamlProperty('windowWidth', yamlValue(d.windowWidth));
		area.appendYamlProperty('windowHeight', yamlValue(d.windowHeight));
		area.appendYamlProperty('colorDepth', yamlValue(d.colorDepth));
		area.appendYamlProperty('pixelRatio', yamlValue(d.pixelRatio));

		area.appendSection('System');
		area.appendYamlProperty('timezone', yamlValue(d.timezone));
		area.appendYamlProperty('timezoneOffsetMinutes', yamlValue(d.timezoneOffset));
		area.appendYamlProperty('cookiesEnabled', yamlValue(d.cookiesEnabled));
		area.appendYamlProperty('localStorage', yamlAvailable(d.localStorageAvailable));
		area.appendYamlProperty('sessionStorage', yamlAvailable(d.sessionStorageAvailable));
		area.appendYamlProperty('indexedDB', yamlAvailable(d.indexedDBAvailable));
		area.appendYamlProperty('secureContext', yamlValue(d.isSecureContext));
		area.appendYamlProperty('online', yamlValue(d.online));

		area.appendSection('Features');
		area.appendYamlProperty('webSocket', yamlSupported(d.webSocketSupported));
		area.appendYamlProperty('webWorkers', yamlSupported(d.webWorkersSupported));
		area.appendYamlProperty('serviceWorkers', yamlSupported(d.serviceWorkersSupported));
		area.appendYamlProperty('fetch', yamlSupported(d.fetchSupported));
		area.appendYamlProperty('geolocation', yamlSupported(d.geolocationSupported));
		area.appendYamlProperty('notifications', yamlSupported(d.notificationsSupported));

		area.appendSection('Performance');
		area.appendYamlProperty('hardwareConcurrency', yamlValue(d.hardwareConcurrency));
		area.appendYamlProperty('deviceMemoryGb', yamlValue(d.deviceMemory));
		area.appendYamlProperty('connectionType', yamlValue(d.connectionType));
		area.appendYamlProperty('connectionEffectiveType', yamlValue(d.effectiveType));

		area.appendSection('PWA');
		area.appendYamlProperty('serviceWorkerRegistered', yamlValue(d.serviceWorkerRegistered));
		area.appendYamlProperty('serviceWorkerVersion', yamlValue(d.serviceWorkerVersion));
		area.appendYamlProperty('serviceWorkerState', yamlValue(d.serviceWorkerState));
		area.appendYamlProperty('serviceWorkerScope', yamlValue(d.serviceWorkerScope));
	}

	async function refreshDiagnostics() {
		loading.value = true;
		errorMessage.value = '';

		try {
			diagnostics.value = await collectDiagnostics();
			await buildDiagnosticsYaml();
		} catch (error) {
			console.error('Error collecting diagnostics:', error);
			errorMessage.value = `Failed to collect browser diagnostics: ${error.message}`;
		} finally {
			loading.value = false;
		}
	}

	onMounted(async () => {
		await nextTick();
		await refreshDiagnostics();
	});
</script>

<style scoped>
	.muted {
		opacity: 0.7;
	}
</style>
