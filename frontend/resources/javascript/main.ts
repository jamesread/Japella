import { createClient } from '@connectrpc/connect'
import { createConnectTransport } from "@connectrpc/connect-web"

import { JapellaControlApiService } from './gen/japella/controlapi/v1/control_pb'

import { createApp } from 'vue';
import Calendar from '../vue/calendar.vue';

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
	createApp(Calendar).mount('#calendar')
}

async function main2(): void {
	const transport = createConnectTransport({
		baseUrl: 'http://localhost:8080/api/',
	})

	const client = createClient(JapellaControlApiService, transport)

	const status = await client.getStatus();

	console.log(status)

	document.getElementById('status').innerText = status.status;
	document.getElementById('nanoservices').innerText = status.nanoservices.join(", ");


//	setupPostBox()
}
