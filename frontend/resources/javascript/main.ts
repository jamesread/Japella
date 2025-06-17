import { createClient } from '@connectrpc/connect'
import { createConnectTransport } from "@connectrpc/connect-web"

import { JapellaControlApiService } from './gen/japella/controlapi/v1/control_pb'

import { createApp } from 'vue';
import { createI18n, useI18n } from 'vue-i18n';
import App from '../vue/App.vue';

import Notification from './notification.js';

function setupPostBox () {
  document.getElementById('submit-post').addEventListener('click', () => {
    submitPost(document.getElementById('post-box').value)
  })
}

function submitPost (post) {
  window.fetch('/api/sendPost', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json'
    },
    body: JSON.stringify({
      post: post
    })
  }).then(response => {
    if (response.ok) {
      document.getElementById('result').value = ''
    }
  })
}

export function main(): void {
	fetch('http://localhost:8080/lang')
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
			console.error('Error loading language file:', error);
		});
}

function createTheApp(i18n: any): void {
	const app = createApp(App)

	app.use(i18n)
	app.mount('#app')

	loadNavSection()

	createApiClient()
	setupApi()

	displayNotifications()
}

function loadNavSection(): void {
	const currentSection = window.location.hash.substring(1);

	if (currentSection === '') {
		showNavSection('welcome');
	} else {
		showNavSection(currentSection);
	}
}

function createSectionHeader(name: string): HTMLHeadingElement {
	const header = document.createElement('h2');
	header.innerText = name;

	document.getElementById('nav-section-links').appendChild(header);

	return header;
}


function showNavSection(sectionClass: string): void {
	document.getElementById('nav-section-links').querySelectorAll('a').forEach((a) => {
			a.classList.remove('active')
	});

	const link = document.querySelector('a[href="#' + sectionClass + '"]');

	if (link) {
		link.classList.add('active')
	}

	for (const section of document.querySelectorAll('section')) {
		const shown = section.classList.contains(sectionClass)

		section.hidden = !shown
	}
}

function createApiClient(): void {
	let baseUrl = '/api/'

	if (window.location.hostname.includes('localhost')) {
		baseUrl = 'http://localhost:8080/api/'
	}

	window.transport = createConnectTransport({
		baseUrl: baseUrl,
	})

	window.client = createClient(JapellaControlApiService, window.transport)

//	setupPostBox()
}

async function setupApi(): void {
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
