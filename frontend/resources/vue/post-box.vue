<template>
	<section class = "post-box">
		<h2>Post</h2>
		<p>Post a message to a service.</p>

		<form @submit.prevent="submitPost" id = "submit-post">
			<label>Type</label>
			<RadioGroup name = "post-mode" v-model = "postMode" :availablePostModes = "availablePostModes" />

			<label>Posting Service</label>
			<div v-if = "postMode == 'canned'">
				<p>Canned posts just get saved.</p>
			</div>
			<div v-else>
				<div v-if="clientReady">
					<div v-if="items.length === 0">
						<InlineNotification type = "note" message = "No posting services available that support wall posting." />
					</div>
					<div v-else>
						<span v-for = "ps in items" :key = "ps.id" class = "check-list">
							<label>
								<Icon :icon="ps.icon" />
								:
								{{ ps.identity }}
								<input type = "checkbox" name = "posting-service" :value = "ps.id" v-model = "selectedService">
							</label>
						</span>
					</div>
				</div>
				<div v-else>
					<p>Loading posting services...</p>
				</div>
			</div>


			<label class = "grid-wide">Message</label>
			<textarea id = "post" rows = "8" cols = "80" class = "grid-wide"></textarea>

			<fieldset>
				<button id = "submit" type = "submit" disabled>Post</button>
			</fieldset>
		</form>
	</section>
</template>

<script setup>
	import { ref, onMounted } from 'vue';
	import RadioGroup from './radio-group.vue';
	import InlineNotification from './inline-notification.vue';
	import { Icon } from '@iconify/vue';

	const clientReady = ref(false);
	const items = ref([]);
	const availablePostModes = ref(['live', 'canned']);

	const waitForClient = () => {
	  return new Promise((resolve) => {
		const check = () => {
		  if (window.client) {
			resolve(true);
		  } else {
			setTimeout(check, 100);
		  }
		};
		check();
	  });
	};

	onMounted(async () => {
	  await waitForClient();
	  clientReady.value = true;

	  const ret = await window.client.getPostingServices()

	  items.value = ret.services
	});
</script>

<script>
	import { ref } from 'vue';

	const postMode = ref('live');

	import { SubmitPostRequestSchema } from './../javascript/gen/japella/controlapi/v1/control_pb'
	import Notification from './../javascript/notification.js'

	function submitPost() {
		const post = document.getElementById('post').value;
		const submit = document.getElementById('submit');

		if (post.length > 0) {
			submit.innerText = "Submitting...";
			submit.disabled = true;

			 // Use `create(SubmitPostRequestSchema)` to create a new message.
			let req = {
				content: post,
				postingService: document.getElementById('posting-service').value,
				service: document.getElementById('posting-service').value
			}

			console.log(req)

			window.client.submitPost(req)
				.then((res) => {
					submit.innerText = "Post";
					submit.disabled = false;

					let n = new Notification("good", "Post Submitted", "Your post has been submitted successfully.")
					n.show()
				})
				.catch((error) => {
					alert("Error posting message: " + error);
					submit.innerText = "Post";
					submit.disabled = false;
				});
		} else {
			alert("Please enter a message.");
		}
	}
</script>
