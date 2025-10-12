import { createClient } from '@connectrpc/connect'
import { createConnectTransport } from "@connectrpc/connect-web"

import { JapellaControlApiService } from './gen/japella/controlapi/v1/control_pb'

import { createApp } from 'vue';
import { createI18n } from 'vue-i18n';
import App from '../vue/App.vue';
import router from './router.js';

import Notification from './notification.js';

export function main(): void {
	fetch('/lang')
		.then(response => response.json())
		.then(ret => {
			const i18n = createI18n({
				legacy: false,
				locale: ret.acceptLanguages[0],
				fallbackLocale: 'en',
				messages: ret.messages,
				postTranslation: (translated) => {
					const params = new URLSearchParams(window.location.search);

					if (params.has('debug-i18n')) {
						return '___'
					} else {
						return translated;
					}
				}
			})

			createTheApp(i18n);
		})
		.catch(error => {
			const errorMessage = document.createElement('p');
			errorMessage.classList.add('bad');
			errorMessage.classList.add('notification');
			errorMessage.style.margin = '2em';
			errorMessage.innerHTML = 'Error loading the initial language file. This could be if the frontend is running, but cannot connect to the Japella server. Please check your browser console for more details. <br /><br />' + error.message;

			document.body.prepend(errorMessage);

			console.error('Error loading language file:', error);
		});
}

function createTheApp(i18n: any): void {
	const app = createApp(App)

	app.use(i18n)
	app.use(router)

	// Make router available globally for backward compatibility
	window.router = router

	app.mount('#app')

	createApiClient()
	//setupApi()

	displayNotifications()
}

function createApiClient(): void {
	window.transport = createConnectTransport({
		baseUrl: '/api/',
		credentials: 'include',
	})

	window.client = createClient(JapellaControlApiService, window.transport)
}

// onLogin function removed - handled by App.vue

function getSearchParams(): URLSearchParams {
	const params = new URLSearchParams(window.location.search);

	return params;
}

function displayNotifications(): void {
	let params = getSearchParams();
	if (params.has('notification')) {
		let type = params.get('type') || 'info';
		let title = params.get('title') || 'Notification';
		let message = params.get('notification');

		let n = new Notification(type, title, message);
		n.show();

		// Clear the notification from the URL
		params.delete('notification');
		window.history.replaceState({}, '', window.location.pathname + '?' + params.toString());
	}
}
