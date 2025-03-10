import { createClient } from '@connectrpc/connect'
import { createConnectionTransport } from "@connectrpc/connect-web"

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

export default function main() {
	setupPostBox()
}
