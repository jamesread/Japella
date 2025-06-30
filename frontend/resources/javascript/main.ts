import { createClient } from '@connectrpc/connect'
import { createConnectTransport } from "@connectrpc/connect-web"

import { JapellaControlApiService } from './gen/japella/controlapi/v1/control_pb'

import { createApp } from 'vue';
import { createI18n, useI18n } from 'vue-i18n';
import App from '../vue/App.vue';

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
			errorMessage.innerHTML = 'Error loading the initial language file, something is badly wrong with the connection to the Japella server. Please check your browser console for more details. <br /><br />' + error.message;

			document.body.prepend(errorMessage);

			console.error('Error loading language file:', error);
		});
}

function createTheApp(i18n: any): void {
	const app = createApp(App)

	app.use(i18n)
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

async function onLogin(): void {
	const status = await window.client.getStatus();

	window.dispatchEvent(new CustomEvent('status-updated', {
		"detail": status
	}));

	document.getElementById('currentVersion').innerText = 'Version: ' + status.version;

}

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
