<script setup>
	import { ref, onMounted } from 'vue';

	const clientReady = ref(false);
	const items = ref([]);

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
	import { SubmitPostRequestSchema } from './../javascript/gen/japella/controlapi/v1/control_pb'

	function submitPost() {
		const post = document.getElementById('post').value;
		const submit = document.getElementById('submit');

		if (post.length > 0) {
			submit.innerText = "Submitting...";
			submit.disabled = true;

 // Use `create(SubmitPostRequestSchema)` to create a new message.
			let req = {
				content: post,
				service: document.getElementById('posting-service').value
			}

			console.log(req)

			window.client.submitPost(req)
				.then((res) => {
					submit.innerText = "Post";
					submit.disabled = false;

					console.log(res);
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
	<section>
		<form @submit.prevent="submitPost">
			<h2>Post</h2>

			<fieldset>
				<div v-if="clientReady">
					<select name = "posting-service" id = "posting-service">
							<option v-for = "ps in items" :key = "ps" value = "{{ ps.name }}">{{ ps.name }}</option>
					</select>
				</div>
				<div v-else>
					<p>Loading posting services...</p>
				</div>
			</fieldset>

			<fieldset>
				<label>Message</label>
				<textarea id = "post" rows = "8" cols = "80"></textarea>
			</fieldset>

			<button id = "submit" type = "submit">Post</button>
		</form>
	</section>
</template>
