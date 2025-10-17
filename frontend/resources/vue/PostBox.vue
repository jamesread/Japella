<template>
    <section class = "post-box small">
        <h2>{{ t('section.postbox.title') }}</h2>
        <p>{{ t('section.postbox.intro') }}</p>

        <form @submit.prevent="submitPost" id = "submit-post">
            <select v-if = "clientReady" v-model = "selectedCampaignId" @change="onCampaignChange" style = "grid-column: 1 / span 2;">
                <option value = "0">Select a campaign...</option>
                <option v-for = "campaign in campaigns" :key = "campaign.id" :value = "campaign.id">{{ campaign.name }}</option>
            </select>
            <div v-else>
                <p>Loading campaigns...</p>
            </div>

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
                                <input type = "checkbox" :id = "ps.id" :name = "ps.id" :value = "ps.id" @change="onPostingServiceChange" />

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



            <textarea ref = "postTextarea" id = "post" rows = "8" cols = "80" class = "gs2" placeholder = "Hello world!" @keyup = "recountLength" @input="recountLength"></textarea>

            <fieldset>
                <button id = "submit" type = "button" class="good" @click="submitPost" :disabled="!canSubmit">Post now</button>
                <button id = "open-schedule" type = "button" class="good" @click="openScheduleDialog" :disabled="!canSchedule">Post Later</button>
            </fieldset>
            <div style = "display: flex; justify-content: flex-end;">
				<span ref = "postLengthCounter">{{ postLength }}</span>
            </div>
            <div v-if="showScheduleDialog" class="modal-overlay" @click.self="cancelScheduleDialog">
                <div class="modal">
                    <h3>Schedule Post</h3>
                    <label for="scheduledAtDialog">Scheduled time</label>
                    <input id="scheduledAtDialog" type="datetime-local" v-model="scheduledAtDialog" />
                    <div class="dialog-actions">
                        <button class="neutral" @click="cancelScheduleDialog">Cancel</button>
                        <button class="good" @click="confirmScheduleDialog" :disabled="!scheduledAtDialog">Schedule</button>
                    </div>
                </div>
            </div>

        </form>
    </section>
</template>

<script setup>
    import { useI18n } from 'vue-i18n'
    const { t } = useI18n()

import { ref, onMounted, onActivated, computed, nextTick } from 'vue';
	import { useRoute } from 'vue-router';
    import { Icon } from '@iconify/vue';
    import InlineNotification from './InlineNotification.vue';

    const clientReady = ref(false);
    const items = ref([]);
	const postLength = ref(0);
const contentLength = ref(0);
	const postLengthCounter = ref(null);
	const postTextarea = ref('');
	const campaigns = ref([]);
	const selectedCampaignId = ref(0);
	const saveAsCannedPost = ref(false);
    const scheduledAt = ref("");
    const showScheduleDialog = ref(false);
    const scheduledAtDialog = ref("");
const selectedServiceCount = ref(0);

	const route = useRoute();

const recountLength = (e) => {
    const length = e.target.value.length;

    postLength.value = length;
    contentLength.value = length;

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

        await refreshAccounts()

		// Check for canned post ID in query parameters
		if (route.query.cannedPostId) {
			await startPost(route.query.cannedPostId);
		}

		// Preselect campaign if passed from navigation
        if (route.query.campaignId) {
			const cid = parseInt(route.query.campaignId, 10);
			if (!isNaN(cid)) {
				selectedCampaignId.value = cid;
                await applyCampaignMembership(cid)
			}
		}
    });

    onActivated(async () => {
        await refreshAccounts()

		// Check for canned post ID in query parameters
		if (route.query.cannedPostId) {
			await startPost(route.query.cannedPostId);
		}

		// Preselect campaign if passed when returning
        if (route.query.campaignId) {
			const cid = parseInt(route.query.campaignId, 10);
			if (!isNaN(cid)) {
				selectedCampaignId.value = cid;
                await applyCampaignMembership(cid)
			}
		}
	})

    async function refreshAccounts() {
      const ret = await window.client.getSocialAccounts({"onlyActive": true});
      const campaignsRet = await window.client.getCampaigns();

      items.value = ret.accounts

      campaigns.value = campaignsRet.campaigns || [];

      if (selectedCampaignId.value && selectedCampaignId.value !== 0) {
    await applyCampaignMembership(selectedCampaignId.value)
      }
  await nextTick()
  updateSelectedServiceCount()
    }

    async function onCampaignChange() {
      if (!selectedCampaignId.value || selectedCampaignId.value === 0) {
        if (postingServiceCheckboxes.value) {
          const boxes = postingServiceCheckboxes.value.querySelectorAll('input[type="checkbox"]');
          for (let x of boxes) {
            if (x.id !== 'canned-post') x.checked = false;
          }
        }
    updateSelectedServiceCount()
        return
      }
  await applyCampaignMembership(selectedCampaignId.value)
  await nextTick()
  updateSelectedServiceCount()
    }

    async function applyCampaignMembership(campaignId) {
      try {
        const res = await window.client.getCampaignSocialAccounts({ campaignId })
        const memberIds = new Set(res.socialAccountIds || [])
        if (!postingServiceCheckboxes.value) return
        const boxes = postingServiceCheckboxes.value.querySelectorAll('input[type="checkbox"]');
        for (let x of boxes) {
          if (x.id === 'canned-post') continue;
          const idNum = Number(x.value)
          x.checked = memberIds.has(idNum)
        }
    await nextTick()
    updateSelectedServiceCount()
      } catch (e) {
        console.error('Failed to fetch campaign memberships', e)
      }
    }

    const postMode = ref('live');

    const postingServiceCheckboxes = ref(null);

    import Notification from './../javascript/notification.js'

function updateSelectedServiceCount() {
  if (!postingServiceCheckboxes.value) { selectedServiceCount.value = 0; return }
  const boxes = postingServiceCheckboxes.value.querySelectorAll('input[type="checkbox"]');
  let count = 0;
  for (let x of boxes) {
    if (x.id === 'canned-post') continue;
    if (x.checked) count++
  }
  selectedServiceCount.value = count;
}

function onPostingServiceChange() {
  updateSelectedServiceCount()
}

const canSubmit = computed(() => contentLength.value > 0 && (selectedServiceCount.value > 0 || saveAsCannedPost.value))
const canSchedule = computed(() => contentLength.value > 0 && selectedServiceCount.value > 0)

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
				contentLength.value = response.post.content.length;
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
                contentLength.value = 0;
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

        if (scheduledAt.value) {
            req.scheduledAt = scheduledAt.value;
        }

        console.log(req)

        window.client.submitPost(req)
            .then((res) => {
                submit.innerText = "Post";
                submit.disabled = false;

                // Clear the textarea
                document.getElementById('post').value = '';
                postLength.value = 0;
                contentLength.value = 0;
                if (postLengthCounter.value) {
                    postLengthCounter.value.classList.remove('bad');
                }

                // Clear scheduling input
                scheduledAt.value = "";

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

    function openScheduleDialog() {
        showScheduleDialog.value = true;
        scheduledAtDialog.value = scheduledAt.value || "";
    }

    function cancelScheduleDialog() {
        showScheduleDialog.value = false;
    }

    function confirmScheduleDialog() {
        scheduledAt.value = scheduledAtDialog.value;
        showScheduleDialog.value = false;
        submitPost();
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
