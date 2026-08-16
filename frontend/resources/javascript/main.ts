import 'picocrank/styles.css';
import './theme.js';

import { createClient } from '@connectrpc/connect'
import { createConnectTransport } from "@connectrpc/connect-web"

import { JapellaControlApiService } from './gen/japella/controlapi/v1/control_pb'
import type { GetStatusResponse } from './gen/japella/controlapi/v1/control_pb'

import { createApp } from 'vue';
import { createI18n } from 'vue-i18n';
import App from '../vue/App.vue';
import router from './router.js';

import { showNotification } from './notifications.js';
import { fetchAppStatus, invalidateAppStatus } from './status.js';

/** Published docs: database readiness and startup failures. */
const DOCS_TROUBLESHOOT_DATABASE =
	'https://jamesread.github.io/Japella/troubleshooting/database-connection.html';
const DOCS_TROUBLESHOOT_SERVER_UNREACHABLE =
	'https://jamesread.github.io/Japella/troubleshooting/server-unreachable.html';

const STARTUP_RETRY_SECONDS = 15;

type HttpError = Error & { status?: number };

type DocLink = { href: string; label: string };

type StartupFailureKind = 'unreachable' | 'database' | 'translations';

class StartupFailure extends Error {
	kind: StartupFailureKind;
	status?: number;

	constructor(kind: StartupFailureKind, message: string, status?: number) {
		super(message);
		this.name = 'StartupFailure';
		this.kind = kind;
		this.status = status;
	}
}

let failureOverlay: HTMLElement | null = null;
let failureRetryTimer: ReturnType<typeof setInterval> | null = null;

function clearStartupFailureUi(): void {
	if (failureRetryTimer != null) {
		clearInterval(failureRetryTimer);
		failureRetryTimer = null;
	}
	failureOverlay?.remove();
	failureOverlay = null;
}

function showStartupFailure(
	summary: string,
	hint: string,
	docLinks?: DocLink[],
	onRetry?: () => Promise<void>,
): void {
	clearStartupFailureUi();

	const overlay = document.createElement('div');
	overlay.className = 'startup-failure-overlay';

	const card = document.createElement('div');
	card.className = 'startup-failure-card bad';

	const title = document.createElement('h1');
	title.textContent = summary;
	card.appendChild(title);

	const hintEl = document.createElement('p');
	hintEl.textContent = hint;
	card.appendChild(hintEl);

	if (docLinks != null && docLinks.length > 0) {
		const list = document.createElement('ul');
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
		card.appendChild(list);
	}

	if (onRetry) {
		const actions = document.createElement('div');
		actions.className = 'startup-failure-actions';

		const retryButton = document.createElement('button');
		retryButton.type = 'button';
		retryButton.className = 'startup-failure-retry';
		retryButton.textContent = 'Retry now';

		const timerEl = document.createElement('p');
		timerEl.className = 'startup-failure-timer';
		const countdownSpan = document.createElement('span');
		let secondsLeft = STARTUP_RETRY_SECONDS;

		const updateCountdown = () => {
			timerEl.textContent = '';
			timerEl.append('Automatically retrying in ');
			countdownSpan.textContent = String(secondsLeft);
			timerEl.append(countdownSpan, 's…');
		};

		updateCountdown();

		let retryInFlight = false;

		const runRetry = async () => {
			if (retryInFlight) {
				return;
			}
			retryInFlight = true;
			if (failureRetryTimer != null) {
				clearInterval(failureRetryTimer);
				failureRetryTimer = null;
			}
			retryButton.disabled = true;
			timerEl.textContent = 'Retrying…';
			try {
				await onRetry();
			} finally {
				retryInFlight = false;
			}
		};

		retryButton.addEventListener('click', () => {
			void runRetry();
		});

		failureRetryTimer = setInterval(() => {
			secondsLeft -= 1;
			if (secondsLeft <= 0) {
				void runRetry();
				return;
			}
			updateCountdown();
		}, 1000);

		actions.append(retryButton, timerEl);
		card.appendChild(actions);
	}

	overlay.appendChild(card);
	document.body.prepend(overlay);
	failureOverlay = overlay;
}

function createApiClient(): void {
	window.transport = createConnectTransport({
		baseUrl: '/api/',
		credentials: 'include',
	})

	window.client = createClient(JapellaControlApiService, window.transport)
}

function isServerUsable(status: GetStatusResponse): boolean {
	return Boolean(status.databaseConnected);
}

async function checkServerReady(force = false): Promise<GetStatusResponse> {
	let status: GetStatusResponse;
	try {
		status = await fetchAppStatus({ force });
	} catch (error: unknown) {
		const message = error instanceof Error ? error.message : String(error);
		throw new StartupFailure('unreachable', message);
	}

	if (!isServerUsable(status)) {
		throw new StartupFailure(
			'database',
			'The Japella server is running but the database is not connected.',
		);
	}

	return status;
}

async function loadLanguage(): Promise<{
	acceptLanguages: string[];
	messages: Record<string, Record<string, string>>;
}> {
	const response = await fetch('/lang');
	if (!response.ok) {
		const err = new Error(`HTTP ${response.status}`) as HttpError;
		err.status = response.status;
		throw err;
	}
	return response.json();
}

function mountApp(lang: {
	acceptLanguages: string[];
	messages: Record<string, Record<string, string>>;
}): void {
	const i18n = createI18n({
		legacy: false,
		locale: lang.acceptLanguages?.[0] || 'en',
		fallbackLocale: 'en',
		messages: lang.messages,
		postTranslation: (translated) => {
			const params = new URLSearchParams(window.location.search);

			if (params.has('debug-i18n')) {
				return '___'
			} else {
				return translated;
			}
		}
	});

	createTheApp(i18n);
}

async function runStartup({ fresh = false }: { fresh?: boolean } = {}): Promise<void> {
	if (fresh) {
		invalidateAppStatus();
	}

	await checkServerReady(fresh);
	const lang = await loadLanguage();
	clearStartupFailureUi();
	mountApp(lang);
}

function scheduleStartupRetry(): () => Promise<void> {
	return async () => {
		try {
			await runStartup({ fresh: true });
		} catch (error: unknown) {
			handleStartupFailure(error);
		}
	};
}

function handleStartupFailure(error: unknown): void {
	const onRetry = scheduleStartupRetry();

	if (error instanceof StartupFailure) {
		if (error.kind === 'database') {
			showStartupFailure(
				'Japella is not ready.',
				'The server responded but the database is not connected. This usually means the database is unreachable or the application has not finished starting.',
				[
					{
						href: DOCS_TROUBLESHOOT_DATABASE,
						label: 'Database connection troubleshooting (documentation)',
					},
				],
				onRetry,
			);
		} else {
			showStartupFailure(
				'Could not reach the Japella server.',
				'The browser could not complete a status check (for example, a network error or the server is unreachable). Check your connection and the browser console.',
				[
					{
						href: DOCS_TROUBLESHOOT_SERVER_UNREACHABLE,
						label: 'Why this happens and how to fix it (documentation)',
					},
				],
				onRetry,
			);
		}
		console.error('Server status check failed:', error);
		return;
	}

	if (error instanceof SyntaxError) {
		showStartupFailure(
			'Translations could not be loaded.',
			'The server response was not valid JSON.',
			undefined,
			onRetry,
		);
		console.error('Error loading language file:', error);
		return;
	}

	const status = (error as HttpError).status;

	if (status === 500) {
		showStartupFailure(
			'Translations could not be loaded.',
			'The Japella server returned an error while loading translations.',
			undefined,
			onRetry,
		);
	} else if (status != null && status >= 400) {
		showStartupFailure(
			'Translations could not be loaded.',
			`The server responded with HTTP ${status} while loading translations. Check that the Japella backend is running and try again.`,
			undefined,
			onRetry,
		);
	} else {
		showStartupFailure(
			'Could not load the Japella web interface.',
			'An unexpected error occurred during startup. Check the browser console for details.',
			undefined,
			onRetry,
		);
	}

	console.error('Startup failed:', error);
}

export async function main(): Promise<void> {
	createApiClient();

	try {
		await runStartup();
	} catch (error: unknown) {
		handleStartupFailure(error);
	}
}

function createTheApp(i18n: any): void {
	const app = createApp(App)

	app.use(i18n)
	app.use(router)

	// Make router available globally for backward compatibility
	window.router = router

	app.mount('#app')

	//setupApi()

	displayNotifications()
	registerServiceWorker()
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

		showNotification(type, title, message);

		// Clear the notification from the URL
		params.delete('notification');
		window.history.replaceState({}, '', window.location.pathname + '?' + params.toString());
	}
}

function registerServiceWorker(): void {
	if (!('serviceWorker' in navigator)) {
		return;
	}

	// Dev (vite / jwrapp): never register — a cached production index.html will
	// request /assets/*.js, which the Vite middlewares 404 for script fetches.
	if (import.meta.env.DEV) {
		navigator.serviceWorker.getRegistrations().then((registrations) => {
			for (const registration of registrations) {
				registration.unregister();
			}
		}).catch(() => { /* ignore */ });
		return;
	}

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
