import { createClient } from '@connectrpc/connect'
import { createConnectTransport } from "@connectrpc/connect-web"

import { JapellaControlApiService } from './gen/japella/controlapi/v1/control_pb'

import { createApp } from 'vue';
import App from '../vue/app.vue';

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
	createApp(App).mount('#app')

	createSectionLink('Social Accounts', 'social-accounts')
	createSectionLink('Canned Posts', 'canned-posts')
	createSectionLink('Post', 'post-box')
//	createSectionLink('Calendar', 'calendar')
//	createSectionLink('Status', 'status')
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

function createSectionLink(name: string, sectionClass: string): HTMLAnchorElement {
	const link = document.createElement('a');
	link.innerText = name;
	link.href = '#' + sectionClass;

	link.onclick = () => showNavSection(sectionClass);

	const li = document.createElement('li');
	li.appendChild(link);

	document.getElementsByTagName('nav')[0].querySelector('ul').appendChild(li);

	return link;
}

function showNavSection(sectionClass: string): void {
	document.getElementsByTagName('nav')[0].querySelector('ul').querySelectorAll('a').forEach((a) => {
			a.classList.remove('active')
	});

	const link = document.querySelector('nav a[href="#' + sectionClass + '"]');

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
