import { createClient } from '@connectrpc/connect'
import { createConnectTransport } from "@connectrpc/connect-web"

import { JapellaControlApiService } from './gen/japella/controlapi/v1/control_pb'

import { createApp } from 'vue';
import Calendar from '../vue/calendar.vue';
import PostBox from '../vue/post-box.vue';
import CannedPosts from '../vue/canned-posts.vue';
import AppStatus from '../vue/app-status.vue';

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
	createApp(PostBox).mount('#post-box')
	createApp(Calendar).mount('#calendar')
	createApp(CannedPosts).mount('#canned-posts')
	createApp(AppStatus).mount('#app-status')

	createSectionLink('Canned Posts', 'canned-posts').click()
	createSectionLink('Post', 'post-box')
	createSectionLink('Calendar', 'calendar')
	createSectionLink('Status', 'status')

	createApiClient()
	setupApi()
}

function createSectionLink(name: string, sectionClass: string): HTMLAnchorElement {
	const link = document.createElement('a');
	link.innerText = name;

	link.onclick = () => {
		document.getElementsByTagName('nav')[0].querySelector('ul').querySelectorAll('a').forEach((a) => {
			a.classList.remove('active')
		});

		for (const section of document.querySelectorAll('section')) {
			const shown = section.classList.contains(sectionClass)

			section.hidden = !shown
		}

		link.classList.add('active')
	}

	const li = document.createElement('li');
	li.appendChild(link);

	document.getElementsByTagName('nav')[0].querySelector('ul').appendChild(li);

	return link;
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
