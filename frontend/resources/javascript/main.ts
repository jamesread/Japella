import 'femtocrank/style.css';
import 'femtocrank/dark.css';

import { createClient } from '@connectrpc/connect'
import { createConnectTransport } from "@connectrpc/connect-web"

import { JapellaControlApiService } from './gen/japella/controlapi/v1/control_pb'

import { createApp } from 'vue';
import { createI18n } from 'vue-i18n';
import App from '../vue/App.vue';
import router from './router.js';

import Notification from './notification.js';

/** Published docs: database readiness and translation load failures. */
const DOCS_TROUBLESHOOT_DATABASE =
	'https://jamesread.github.io/Japella/troubleshooting/database-connection.html';

type HttpError = Error & { status?: number };

type DocLink = { href: string; label: string };

function showLanguageLoadFailure(summary: string, hint: string, docLinks?: DocLink[]): void {
	const block = document.createElement('div');
	block.classList.add('bad');
	block.classList.add('notification');
	block.style.margin = '2em';

	const title = document.createElement('strong');
	title.textContent = summary;
	block.appendChild(title);
	block.appendChild(document.createElement('br'));
	block.appendChild(document.createElement('br'));

	const hintEl = document.createElement('p');
	hintEl.textContent = hint;
	hintEl.style.margin = '0 0 0.75em 0';
	block.appendChild(hintEl);

	if (docLinks != null && docLinks.length > 0) {
		const list = document.createElement('ul');
		list.style.margin = '0';
		list.style.paddingLeft = '1.25em';
		for (const { href, label } of docLinks) {
			const li = document.createElement('li');
			const a = document.createElement('a');
			a.href = href;
			a.target = '_blank';
			a.rel = 'noopener noreferrer';
			a.textContent = label;
			li.appendChild(a);
			list.appendChild(li);
		}
		block.appendChild(list);
	}

	document.body.prepend(block);
}

export function main(): void {
	fetch('/lang')
		.then(async (response) => {
			if (!response.ok) {
				const err = new Error(`HTTP ${response.status}`) as HttpError;
				err.status = response.status;
				throw err;
			}
			return response.json();
		})
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
		.catch((error: unknown) => {
			if (error instanceof SyntaxError) {
				showLanguageLoadFailure(
					'Translations could not be loaded.',
					'The server response was not valid JSON.',
				);
				console.error('Error loading language file:', error);
				return;
			}

			const status = (error as HttpError).status;

			if (status === 500) {
				showLanguageLoadFailure(
					'Translations could not be loaded.',
					'The Japella server returned an error while loading translations. This usually means the database is not reachable or the application has not finished starting.',
					[
						{
							href: DOCS_TROUBLESHOOT_DATABASE,
							label: 'Database connection troubleshooting (documentation)',
						},
					],
				);
			} else if (status != null && status >= 400) {
				showLanguageLoadFailure(
					'Translations could not be loaded.',
					`The server responded with HTTP ${status}. Check that the Japella backend is running and try again.`,
				);
			} else {
				showLanguageLoadFailure(
					'Could not reach the Japella server to load translations.',
					'The browser could not complete the request (for example, a network error or the server is unreachable). Check your connection and the browser console.',
				);
			}

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
	registerServiceWorker()
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

function registerServiceWorker(): void {
	if ('serviceWorker' in navigator) {
		window.addEventListener('load', () => {
			navigator.serviceWorker.register('/sw.js')
				.then((registration) => {
					console.log('[Service Worker] Registration successful:', registration.scope);

					// Check for updates periodically
					setInterval(() => {
						registration.update();
					}, 60000); // Check every minute
				})
				.catch((error) => {
					console.error('[Service Worker] Registration failed:', error);
				});
		});
	}
}
