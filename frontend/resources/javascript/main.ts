import { createClient } from '@connectrpc/connect'
import { createConnectTransport } from "@connectrpc/connect-web"

import { JapellaControlApiService } from './gen/japella/controlapi/v1/control_pb'

import { createApp } from 'vue';
import Calendar from '../vue/calendar.vue';
import PostBox from '../vue/post-box.vue';

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

	createApiClient()
	setupApi()
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

	document.getElementById('status').innerText = status.status;
	document.getElementById('nanoservices').innerText = status.nanoservices.join(", ") + "(" + (status.nanoservices.length > 0 ? "" : "") + status.nanoservices.length + " nanoservices)";
	document.getElementById('currentVersion').innerText = 'Version: ' + status.version;

}
