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

            <div v-if="postMode != 'canned'" class="media-attach">
                <label>Attach media</label>
                <div v-if="selectedMedia.length" class="selected-media">
                    <div v-for="(m, idx) in selectedMedia" :key="m.url" class="selected-media-item">
                        <img v-if="isImageFile(m.filename)" :src="m.url" :alt="m.filename" class="selected-media-thumb" />
                        <span v-else class="selected-media-placeholder">{{ m.filename }}</span>
                        <button type="button" class="remove-media" @click="removeMedia(idx)" title="Remove">×</button>
                    </div>
                </div>
                <button type="button" class="neutral add-media-btn" @click="toggleMediaPicker">
                    {{ showMediaPicker ? 'Close library' : 'Add from library' }}
                </button>
                <div v-if="showMediaPicker" class="media-picker">
                    <p v-if="!mediaLibrary.length">No media in library. <router-link to="/media">Upload some</router-link> first.</p>
                    <div v-else class="media-picker-grid">
                        <button type="button" v-for="item in mediaLibrary" :key="item.url" class="media-picker-item" @click="addMedia(item)">
                            <img v-if="isImageFile(item.filename)" :src="item.url" :alt="item.filename" />
                            <span v-else>{{ item.filename }}</span>
                        </button>
                    </div>
                </div>
            </div>

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
	const selectedMedia = ref([]);
	const mediaLibrary = ref([]);
	const showMediaPicker = ref(false);

	const route = useRoute();

	function isImageFile(filename) {
		const ext = (filename || '').split('.').pop()?.toLowerCase();
		return ['jpg', 'jpeg', 'png', 'gif', 'webp', 'svg', 'bmp', 'ico'].includes(ext);
	}
	async function loadMediaLibrary() {
		try {
			const res = await window.client.listMedia({});
			mediaLibrary.value = res.items ?? [];
		} catch (e) {
			mediaLibrary.value = [];
		}
	}
	function toggleMediaPicker() {
		showMediaPicker.value = !showMediaPicker.value;
		if (showMediaPicker.value && mediaLibrary.value.length === 0) loadMediaLibrary();
	}
	function addMedia(item) {
		selectedMedia.value = [...selectedMedia.value, item];
	}
	function removeMedia(index) {
		selectedMedia.value = selectedMedia.value.filter((_, i) => i !== index);
	}

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

                // Clear the textarea and selected media
                document.getElementById('post').value = '';
                postLength.value = 0;
                contentLength.value = 0;
                selectedMedia.value = [];
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
        if (selectedMedia.value.length) {
            req.mediaUrls = selectedMedia.value.map(m => m.url);
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

                // Clear scheduling input and selected media
                scheduledAt.value = "";
                selectedMedia.value = [];

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

	.check-list {
		display: inline-block;
		margin: 0.25em 0.5em 0.25em 0;
	}

	.check-list label {
		cursor: pointer;
		display: inline-block;
	}

	.check-list label > span {
		white-space: nowrap;
		border-radius: 0.5em;
		background-color: #f5f5f5;
		color: #333;
		border: 1px solid #e0e0e0;
		font-weight: 500;
		padding: 0.4em 0.6em;
		display: inline-flex;
		align-items: center;
		gap: .4em;
		user-select: none;
		transition: all 0.2s ease;
		box-shadow: 0 1px 2px rgba(0, 0, 0, 0.05);
	}

	.check-list input[type="checkbox"]:checked + span {
		background-color: #e3f2fd;
		border-color: #2196f3;
		color: #1565c0;
		box-shadow: 0 2px 4px rgba(33, 150, 243, 0.2);
	}

	.check-list label:hover > span {
		background-color: #e8e8e8;
		border-color: #d0d0d0;
		box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
		transform: translateY(-1px);
	}

	.check-list input[type="checkbox"]:checked + span:hover {
		background-color: #bbdefb;
		border-color: #1976d2;
		box-shadow: 0 2px 4px rgba(33, 150, 243, 0.3);
	}

	@media (prefers-color-scheme: dark) {
		.check-list label > span {
			background-color: #2a2a2a;
			color: #e0e0e0;
			border-color: #404040;
			box-shadow: 0 1px 2px rgba(0, 0, 0, 0.3);
		}

		.check-list input[type="checkbox"]:checked + span {
			background-color: #1e3a5f;
			border-color: #2196f3;
			color: #90caf9;
			box-shadow: 0 2px 4px rgba(33, 150, 243, 0.3);
		}

		.check-list label:hover > span {
			background-color: #353535;
			border-color: #505050;
			box-shadow: 0 2px 4px rgba(0, 0, 0, 0.4);
		}

		.check-list input[type="checkbox"]:checked + span:hover {
			background-color: #2a4a6f;
			border-color: #42a5f5;
			box-shadow: 0 2px 4px rgba(33, 150, 243, 0.4);
		}
	}

	.media-attach {
		margin-top: 0.75em;
	}
	.media-attach label {
		display: block;
		font-weight: 500;
		margin-bottom: 0.25em;
	}
	.selected-media {
		display: flex;
		flex-wrap: wrap;
		gap: 0.5em;
		margin-bottom: 0.5em;
	}
	.selected-media-item {
		position: relative;
		width: 64px;
		height: 64px;
		border-radius: 0.5em;
		overflow: hidden;
		border: 1px solid var(--border-color, #ccc);
		background: var(--card-bg, #f0f0f0);
	}
	.selected-media-thumb {
		width: 100%;
		height: 100%;
		object-fit: cover;
		display: block;
	}
	.selected-media-placeholder {
		font-size: 0.65em;
		padding: 0.25em;
		word-break: break-all;
		display: block;
	}
	.remove-media {
		position: absolute;
		top: 2px;
		right: 2px;
		width: 22px;
		height: 22px;
		padding: 0;
		border-radius: 50%;
		background: rgba(0,0,0,0.6);
		color: #fff;
		border: none;
		cursor: pointer;
		font-size: 1.1em;
		line-height: 1;
	}
	.add-media-btn {
		margin-top: 0.25em;
	}
	.media-picker {
		margin-top: 0.75em;
		padding: 0.75em;
		border: 1px solid var(--border-color, #ccc);
		border-radius: 0.5em;
		background: var(--card-bg, #f8f8f8);
		max-height: 200px;
		overflow-y: auto;
	}
	.media-picker-grid {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(72px, 1fr));
		gap: 0.5em;
	}
	.media-picker-item {
		aspect-ratio: 1;
		border-radius: 0.5em;
		border: 1px solid var(--border-color, #ccc);
		background: #fff;
		cursor: pointer;
		padding: 0;
		overflow: hidden;
		display: flex;
		align-items: center;
		justify-content: center;
	}
	.media-picker-item img {
		width: 100%;
		height: 100%;
		object-fit: cover;
	}
	.media-picker-item span {
		font-size: 0.7em;
		word-break: break-all;
		text-align: center;
		padding: 0.25em;
	}
</style>
