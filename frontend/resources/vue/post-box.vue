<script setup>
	import { ref, onMounted } from 'vue';
	import RadioGroup from './radio-group.vue';

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

<style lang = "css" scoped>
fieldset {
	flex-direction: column;
}
</style>

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
					<select name = "posting-service" id = "posting-service">
						<option v-for = "ps in items" :key = "ps" :value = "ps.id">{{ ps.protocol }}: {{ ps.identity }}</option>
					</select>
				</div>
				<div v-else>
					<p>Loading posting services...</p>
				</div>
			</div>


			<label class = "grid-wide">Message</label>
			<textarea id = "post" rows = "8" cols = "80" class = "grid-wide"></textarea>

			<fieldset>
				<button id = "submit" type = "submit">Post</button>
			</fieldset>
		</form>
	</section>
</template>
