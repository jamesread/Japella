<template>
	<section class = "post-box small">
		<h2>Post</h2>
		<p>Post a message to a service.</p>

		<form @submit.prevent="submitPost" id = "submit-post">
			<label>Type</label>
			<RadioGroup name = "post-mode" v-model = "postMode" :availablePostModes = "availablePostModes" />

			<span class = "fake-label">Social account</span>
			<div v-if = "postMode == 'canned'">
				<p>Canned posts just get saved.</p>
			</div>
			<div v-else>
				<div v-if="clientReady">
					<div v-if="items.length === 0">
						<InlineNotification type = "note" message = "No posting services available that support wall posting." />
					</div>
					<div v-else ref = "postingServiceCheckboxes">
						<span v-for = "ps in items" :key = "ps.id" class = "check-list">
							<label>
								<input type = "checkbox" :id = "ps.id" :name = "ps.id" :value = "ps.id" />

								<span>
								<Icon :icon="ps.icon" />
								{{ ps.identity }}
								</span>
							</label
							>
						</span>
					</div>
				</div>
				<div v-else>
					<p>Loading social accounts...</p>
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

<script setup>
	import { ref, onMounted } from 'vue';
	import RadioGroup from './radio-group.vue';
	import InlineNotification from './inline-notification.vue';
	import { Icon } from '@iconify/vue';

	const clientReady = ref(false);
	const items = ref([]);
	const availablePostModes = ref(['live', 'canned', 'scheduled']);

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

	  const ret = await window.client.getSocialAccounts();

	  items.value = ret.accounts
	});
</script>

<script>
	import { ref } from 'vue';

	const postMode = ref('live');

	const postingServiceCheckboxes = ref(null);

	import { SubmitPostRequestSchema } from './../javascript/gen/japella/controlapi/v1/control_pb'
	import Notification from './../javascript/notification.js'

	function getPostingServices() {
		const boxes = postingServiceCheckboxes.value.querySelectorAll('input[type="checkbox"]');

		let ret = [];

		for (let x of boxes) {
			if (x.checked) {
				ret.push(x.id);
			}
		}

		console.log("Boxes: ", ret)
		return ret;
	}

	function submitPost() {
		const post = document.getElementById('post').value;
		const submit = document.getElementById('submit');

		if (post.length == 0) {
			alert("Please enter a message.");
			return;
		}
		submit.innerText = "Submitting...";
		submit.disabled = true;

		 // Use `create(SubmitPostRequestSchema)` to create a new message.
		let req = {
			content: post,
			socialAccounts: getPostingServices(),
		}

		console.log(req)

		window.client.submitPost(req)
			.then((res) => {
				submit.innerText = "Post";
				submit.disabled = false;

				let status = ""

				if (res.success) {
					status = "good"
				} else {
					status = "bad"
				}

				let n = new Notification(status, "Post Submitted", "Your post has been submitted successfully.", res.postUrl)

				n.show()
			})
			.catch((error) => {
				alert("Error posting message: " + error);
				submit.innerText = "Post";
				submit.disabled = false;
			});
	}
</script>
