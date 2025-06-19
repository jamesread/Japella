<template>
	<section class = "post-box small">
		<h2>{{ t('section.postbox.title') }}</h2>
		<p>{{ t('section.postbox.intro') }}</p>

		<form @submit.prevent="submitPost" id = "submit-post">
			<span class = "fake-label">{{ t('section.postbox.socialaccounts') }}</span>
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


			<textarea id = "post" rows = "8" cols = "80" class = "gs2" placeholder = "Hello world!"></textarea>

			<fieldset>
				<button id = "submit" type = "submit">{{ t('section.postbox.submit') }}</button>
			</fieldset>
		</form>
	</section>
</template>

<script setup>
	import { useI18n } from 'vue-i18n'
	const { t } = useI18n()

	import { ref, onMounted } from 'vue';
	import { Icon } from '@iconify/vue';

	const clientReady = ref(false);
	const items = ref([]);
	const availablePostModes = ref([
		{
			"title": "Live",
			"disabled": false,
		},
		{
			"title": "Canned",
			"disabled": true,
		},
		{
			"title": "Scheduled",
			"disabled": true,
		}
	]);

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

	  const ret = await window.client.getSocialAccounts({"onlyActive": true});

	  items.value = ret.accounts
	});
</script>

<script>
	import { ref } from 'vue';

	const postMode = ref('live');

	const postingServiceCheckboxes = ref(null);

	import { SubmitPostRequestSchema } from './../javascript/gen/japella/controlapi/v1/control_pb'
	import Notification from './../javascript/notification.js'

	function getSelectedPostingServices() {
		const boxes = postingServiceCheckboxes.value.querySelectorAll('input[type="checkbox"]');

		let ret = [];

		for (let x of boxes) {
			if (x.checked) {
				ret.push(Number(x.value));
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
			socialAccounts: getSelectedPostingServices(),
		}

		console.log(req)

		window.client.submitPost(req)
			.then((res) => {
				submit.innerText = "Post";
				submit.disabled = false;

				for (let x of res.posts) {
					let status = ""

					if (x.success) {
						status = "good"
					} else {
						status = "bad"
					}

					let n = new Notification(status, "Posted to" + x.socialAccountIcon, "Post complete.", x.postUrl)
					n.show()
				}
			})
			.catch((error) => {
				alert("Error posting message: " + error);
				submit.innerText = "Post";
				submit.disabled = false;
			});
	}
</script>
