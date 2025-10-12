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

                        <!-- Canned Post Option within social accounts -->
                        <span class="check-list">
                            <label>
                                <input type="checkbox" id="canned-post" name="canned-post" value="canned-post" v-model="saveAsCannedPost" />
                                <span>
                                    <Icon icon="jam:box" />
                                    Canned post
                                </span>
                            </label>
                        </span>
                    </div>
                </div>
                <div v-else>
                    <p>Loading social accounts...</p>
                </div>
            </div>

			<label>Campaign ID</label>
			<select v-if = "clientReady" v-model = "selectedCampaignId">
				<option value = "0">None</option>
				<option v-for = "campaign in campaigns" :key = "campaign.id" :value = "campaign.id">{{ campaign.name }}</option>
			</select>
			<div v-else>
				<p>Loading campaigns...</p>
			</div>

            <textarea ref = "postTextarea" id = "post" rows = "8" cols = "80" class = "gs2" placeholder = "Hello world!" @keyup = "recountLength" ></textarea>

            <fieldset>
                <button id = "submit" type = "submit">{{ t('section.postbox.submit') }}</button>
				<div class = "fg1"></div>
				<span ref = "postLengthCounter">{{ postLength }}</span>
            </fieldset>
        </form>
    </section>
</template>

<script setup>
    import { useI18n } from 'vue-i18n'
    const { t } = useI18n()

    import { ref, onMounted, onActivated } from 'vue';
    import { useRoute } from 'vue-router';
    import { Icon } from '@iconify/vue';
    import InlineNotification from './InlineNotification.vue';

    const clientReady = ref(false);
    const items = ref([]);
	const postLength = ref(0);
	const postLengthCounter = ref(null);
	const postTextarea = ref('');
	const campaigns = ref([]);
	const selectedCampaignId = ref(0);
	const saveAsCannedPost = ref(true);

	const route = useRoute();

	const recountLength = (e) => {
	    const length = e.target.value.length;

		postLength.value = post.value.length;

		if (length > 280) {
			postLengthCounter.value.classList.add('bad');
		} else {
			postLengthCounter.value.classList.remove('bad');
		}
	}

	/*
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
	*/

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

		refreshAccounts()

		// Check for canned post ID in query parameters
		if (route.query.cannedPostId) {
			await startPost(route.query.cannedPostId);
		}
    });

	onActivated(async () => {
		refreshAccounts()

		// Check for canned post ID in query parameters
		if (route.query.cannedPostId) {
			await startPost(route.query.cannedPostId);
		}
	})

	async function refreshAccounts() {
      const ret = await window.client.getSocialAccounts({"onlyActive": true});
	  const campaignsRet = await window.client.getCampaigns();

      items.value = ret.accounts

	  campaigns.value = campaignsRet.campaigns || [];
	}

    const postMode = ref('live');

    const postingServiceCheckboxes = ref(null);

    import Notification from './../javascript/notification.js'

	async function startPost(cannedPostId) {
		if (!cannedPostId) {
			console.log('No canned post ID provided');
			return;
		}

		try {
			// Convert string ID to number for the API call
			const id = parseInt(cannedPostId, 10);
			if (isNaN(id)) {
				console.error('Invalid canned post ID:', cannedPostId);
				return;
			}

			// Fetch the canned post by ID
			const response = await window.client.getCannedPost({ id: id });
			if (response.post && response.post.content) {
				postTextarea.value.value = response.post.content;
				// Update the character count
				postLength.value = response.post.content.length;
				if (postLengthCounter.value) {
					if (response.post.content.length > 280) {
						postLengthCounter.value.classList.add('bad');
					} else {
						postLengthCounter.value.classList.remove('bad');
					}
				}
			} else {
				console.error('Canned post not found or has no content');
			}
		} catch (error) {
			console.error('Error fetching canned post:', error);
		}
	}

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

    function submitPost(x) {
        const post = document.getElementById('post').value;
        const submit = document.getElementById('submit');

        if (post.length == 0) {
            alert("Please enter a message.");
            return;
        }
        submit.innerText = "Submitting...";
        submit.disabled = true;

        const selectedServices = getSelectedPostingServices();

        // If no social accounts are selected OR canned post option is checked, create a canned post
        if (selectedServices.length === 0 || saveAsCannedPost.value) {
            window.client.createCannedPost({
                content: post,
                campaignId: selectedCampaignId.value,
            })
            .then((res) => {
                submit.innerText = "Post";
                submit.disabled = false;

                // Clear the textarea
                document.getElementById('post').value = '';
                postLength.value = 0;
                if (postLengthCounter.value) {
                    postLengthCounter.value.classList.remove('bad');
                }

                // Show notification with link to canned posts
                let n = new Notification("good", "Canned Post Created",
                    `Your post has been saved as a canned post. Click here to view it.`,
                    "/canned-posts");
                n.show();
            })
            .catch((error) => {
                alert("Error creating canned post: " + error);
                submit.innerText = "Post";
                submit.disabled = false;
            });
            return;
        }

         // Use `create(SubmitPostRequestSchema)` to create a new message.
        let req = {
            content: post,
            socialAccounts: selectedServices,
			campaignId: selectedCampaignId.value,
        }

        console.log(req)

        window.client.submitPost(req)
            .then((res) => {
                submit.innerText = "Post";
                submit.disabled = false;

                // Clear the textarea
                document.getElementById('post').value = '';
                postLength.value = 0;
                if (postLengthCounter.value) {
                    postLengthCounter.value.classList.remove('bad');
                }

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

	defineExpose({
		startPost,
	});
</script>

<style scoped>
    input[type="checkbox"] {
        display: none;
    }

	fieldset {
        align-items: center;
	}
</style>
