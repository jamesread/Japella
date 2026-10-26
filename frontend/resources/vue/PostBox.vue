<template>
    <div class="post-compose-stack small">
    <section class = "post-box" :class="{ transparent: embedded, embedded }">
        <h2 v-if="!embedded">{{ t('section.postbox.title') }}</h2>
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
                <SocialAccountsSelector
                    v-if="clientReady"
                    v-model="selectedAccountIds"
                    v-model:canned-post="saveAsCannedPost"
                    :only-active="true"
                    :only-can-post="true"
                    :show-canned-post-option="true"
                    empty-message="No posting services available that support wall posting."
                    @loaded="onAccountsLoaded"
                />
                <div v-else>
                    <p>Loading social accounts...</p>
                </div>
            </div>



            <div class="post-composer">
                <textarea ref = "postTextarea" id = "post" rows = "8" class = "post-composer-input" placeholder = "Hello world!" @keyup = "recountLength" @input="recountLength"></textarea>

                <div class="post-composer-meta">
                    <div class="post-composer-tools">
                        <button
                            type="button"
                            class="neutral composer-tool-btn emoji-picker-trigger"
                            :aria-label="t('section.postbox.emoji')"
                            :aria-expanded="showEmojiPicker"
                            @click.stop="toggleEmojiPicker"
                        >
                            <Icon icon="mdi:emoticon-happy-outline" />
                        </button>
                        <EmojiPickerPopover
                            :open="showEmojiPicker"
                            @select="insertEmoji"
                            @close="showEmojiPicker = false"
                        />
                        <button
                            v-if="postMode != 'canned'"
                            type="button"
                            class="neutral composer-tool-btn"
                            :class="{ active: showMediaPicker }"
                            :aria-label="showMediaPicker ? t('section.postbox.closeMedia') : t('section.postbox.addMedia')"
                            :aria-expanded="showMediaPicker"
                            @click="toggleMediaPicker"
                        >
                            <Icon icon="mdi:image-plus" />
                        </button>
                        <button
                            v-if="postMode != 'canned'"
                            type="button"
                            class="neutral composer-tool-btn"
                            :class="{ active: scheduledAt || showScheduleDialog }"
                            :aria-label="t('section.postbox.schedule')"
                            :aria-expanded="showScheduleDialog"
                            @click="openScheduleDialog"
                        >
                            <Icon icon="mdi:calendar-clock" />
                        </button>
                        <div
                            v-if="postMode != 'canned' && enabledLlmTools.length"
                            class="llm-tools-menu-wrap"
                        >
                            <button
                                type="button"
                                class="neutral composer-tool-btn llm-tools-menu-trigger"
                                :class="{ active: showLlmToolsMenu || pendingLlmRunCount > 0 }"
                                aria-label="LLM tools"
                                :aria-expanded="showLlmToolsMenu"
                                :aria-haspopup="true"
                                @click.stop="toggleLlmToolsMenu"
                            >
                                <Icon icon="mdi:creation-outline" />
                            </button>
                            <div
                                v-if="showLlmToolsMenu"
                                ref="llmToolsMenuEl"
                                class="llm-tools-menu"
                                role="menu"
                                aria-label="LLM tools"
                            >
                                <p v-if="contentLength === 0" class="llm-tools-menu-hint">Add post content to run a tool.</p>
                                <button
                                    v-for="tool in enabledLlmTools"
                                    :key="tool.id"
                                    type="button"
                                    class="llm-tools-menu-item"
                                    role="menuitem"
                                    :disabled="contentLength === 0"
                                    :title="tool.description || tool.name"
                                    @click="onSelectLlmTool(tool.id)"
                                >
                                    <span class="llm-tools-menu-item-name">{{ tool.name }}</span>
                                    <span v-if="tool.description" class="llm-tools-menu-item-desc">{{ tool.description }}</span>
                                </button>
                            </div>
                        </div>
                    </div>
                    <span ref = "postLengthCounter" class="post-length-counter">{{ postLength }}</span>
                </div>

                <p v-if="scheduledAt" class="schedule-summary">
                    <Icon icon="mdi:calendar-clock" />
                    <span>{{ formatHumanDateTime(scheduledAt) }}</span>
                    <button type="button" class="schedule-clear" :aria-label="t('section.postbox.clearSchedule')" @click="clearSchedule">×</button>
                </p>

                <div v-if="postMode != 'canned' && (showMediaPicker || selectedMedia.length)" class="media-attach">
                    <div v-if="selectedMedia.length" class="selected-media">
                        <div v-for="(m, idx) in selectedMedia" :key="m.url" class="selected-media-item">
                            <img v-if="isImageFile(m.filename)" :src="m.url" :alt="m.filename" class="selected-media-thumb" />
                            <span v-else class="selected-media-placeholder">{{ m.filename }}</span>
                            <button type="button" class="remove-media" @click="removeMedia(idx)" title="Remove">×</button>
                            <span
                                v-if="mediaVerdict(m) !== 'ok'"
                                class="tag media-verdict"
                                :class="VERDICT_KARMA[mediaVerdict(m)]"
                                :title="mediaVerdictTitle(m)"
                            >
                                <Icon :icon="VERDICT_ICONS[mediaVerdict(m)]" aria-hidden="true" />
                            </span>
                        </div>
                    </div>
                    <ul v-if="optimisedSwaps.length" class="media-swaps" aria-label="Optimised copies available">
                        <li v-for="swap in optimisedSwaps" :key="swap.index">
                            <Icon icon="mdi:image-auto-adjust" aria-hidden="true" />
                            <span>
                                <strong>{{ swap.original.filename }}</strong> has
                                {{ swap.optimised.optimisedKind === 'posting' ? 'a copy prepared for posting' : 'a lossless copy that looks identical' }}
                                ({{ formatFileSize(swap.optimised.sizeBytes) }} instead of {{ formatFileSize(swap.original.sizeBytes) }}).
                            </span>
                            <button type="button" class="neutral small" @click="useOptimised(swap)">Use it</button>
                        </li>
                    </ul>
                    <ul v-if="mediaProblems.length" class="media-problems" aria-label="Media suitability for the chosen accounts">
                        <li v-for="p in mediaProblems" :key="`${p.filename}-${p.network.protocol}`">
                            <span class="tag media-problem-verdict" :class="VERDICT_KARMA[p.network.verdict]">
                                <Icon :icon="p.network.icon" aria-hidden="true" />
                                <Icon :icon="VERDICT_ICONS[p.network.verdict]" aria-hidden="true" />
                            </span>
                            <span><strong>{{ p.filename }}</strong> on {{ describeNetwork(p.network) }}</span>
                        </li>
                    </ul>
                    <div v-if="showMediaPicker" class="media-picker">
                        <p v-if="!mediaLibrary.length">No media in library. <router-link to="/media">Upload some</router-link> first.</p>
                        <div v-else class="media-picker-grid">
                            <button
                                type="button"
                                v-for="item in mediaLibrary"
                                :key="item.url"
                                class="media-picker-item"
                                :title="mediaVerdictTitle(item) || item.filename"
                                @click="addMedia(item)"
                            >
                                <img v-if="isImageFile(item.filename)" :src="item.url" :alt="item.filename" />
                                <span v-else>{{ item.filename }}</span>
                                <span
                                    v-if="mediaVerdict(item) !== 'ok'"
                                    class="tag media-verdict"
                                    :class="VERDICT_KARMA[mediaVerdict(item)]"
                                >
                                    <Icon :icon="VERDICT_ICONS[mediaVerdict(item)]" aria-hidden="true" />
                                </span>
                            </button>
                        </div>
                    </div>
                </div>

                <fieldset class="post-actions">
                    <button id="save-draft" type="button" class="neutral" @click="saveDraft" :disabled="!canSaveDraft">Save draft</button>
                    <button id="submit" type="button" class="good" @click="submitPost" :disabled="!canSubmit">{{ submitButtonLabel }}</button>
                </fieldset>
            </div>

            <div v-if="showScheduleDialog" class="modal-overlay" @click.self="cancelScheduleDialog">
                <div class="modal">
                    <h3>Schedule Post</h3>
                    <label for="scheduledAtDialog">Scheduled time</label>
                    <input id="scheduledAtDialog" type="datetime-local" v-model="scheduledAtDialog" />
                    <div class="dialog-actions">
                        <button v-if="scheduledAtDialog" type="button" class="neutral" @click="clearScheduleDialog">{{ t('section.postbox.clearSchedule') }}</button>
                        <button type="button" class="neutral" @click="cancelScheduleDialog">{{ t('section.postbox.cancel') }}</button>
                        <button type="button" class="good" @click="confirmScheduleDialog" :disabled="!scheduledAtDialog">{{ t('section.postbox.setSchedule') }}</button>
                    </div>
                </div>
            </div>

        </form>
    </section>

    <LlmToolRunsPanel
        v-if="postMode != 'canned' && (llmToolSessions.length || llmToolError)"
        :sessions="llmToolSessions"
        :tools="enabledLlmTools"
        :can-run="false"
        :can-rerun="contentLength > 0"
        :loading="llmToolsLoading"
        :error-message="llmToolError"
        :show-runner="false"
        :flush="true"
        @rerun="runLlmTool"
    />
    </div>
</template>

<script setup>
	defineProps({
		// Shown inside a dialog: the dialog supplies the title and chrome, so the
		// section drops its own card styling and heading.
		embedded: {
			type: Boolean,
			default: false,
		},
	});
    import { useI18n } from 'vue-i18n'
    const { t } = useI18n()

import { ref, onMounted, onActivated, onUnmounted, computed, nextTick, watch } from 'vue';
	import { useRoute, useRouter } from 'vue-router';
    import { Icon } from '@iconify/vue';
    import EmojiPickerPopover from './EmojiPickerPopover.vue';
    import SocialAccountsSelector from './SocialAccountsSelector.vue';
    import LlmToolRunsPanel from './LlmToolRunsPanel.vue';
    import { insertAtCursor, formatHumanDateTime, formatFileSize } from '../javascript/util.js';
    import { buildLlmToolSessions, replyFromApiRun } from '../javascript/llmToolSessions.js';
    import { takeComposePrefill } from '../javascript/composePrefill.js';
    import { VERDICT_ICONS, VERDICT_KARMA, worstVerdict, describeNetwork } from '../javascript/mediaSuitability.js';

    const clientReady = ref(false);
    const items = ref([]);
	const postLength = ref(0);
const contentLength = ref(0);
	const postLengthCounter = ref(null);
	const postTextarea = ref('');
	const campaigns = ref([]);
	const selectedCampaignId = ref(0);
	const selectedAccountIds = ref([]);
	const saveAsCannedPost = ref(false);
    const scheduledAt = ref("");
    const showScheduleDialog = ref(false);
    const scheduledAtDialog = ref("");
	const selectedMedia = ref([]);
	const mediaLibrary = ref([]);
	const showMediaPicker = ref(false);
	const showEmojiPicker = ref(false);
	const showLlmToolsMenu = ref(false);
	const llmToolsMenuEl = ref(null);
	const draftPostId = ref(0);
	const enabledLlmTools = ref([]);
	const llmToolSessions = ref([]);
	const llmToolsLoading = ref(false);
	const llmToolError = ref('');
	let llmElapsedTimer = null;
	let draftEnsurePromise = null;
	/** @type {number[]} */
	let pendingPrefillAccountIds = [];

	const route = useRoute();
	const router = useRouter();

	const pendingLlmRunCount = computed(() =>
		llmToolSessions.value.filter((s) => s.pending).length,
	);

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
		if (showMediaPicker.value) {
			showEmojiPicker.value = false;
			showLlmToolsMenu.value = false;
		}
		if (showMediaPicker.value && mediaLibrary.value.length === 0) loadMediaLibrary();
	}
	function addMedia(item) {
		selectedMedia.value = [...selectedMedia.value, item];
	}

	// Suitability checks per media URL, from the library listing or, for media
	// added by link (e.g. "Use in new post"), fetched one at a time.
	const mediaSuitability = ref({});

	watch(mediaLibrary, (library) => {
		const next = { ...mediaSuitability.value };
		for (const item of library) {
			next[item.url] = item.suitability || [];
		}
		mediaSuitability.value = next;
	});

	watch(selectedMedia, async (media) => {
		for (const m of media) {
			if (mediaSuitability.value[m.url] || !m.filename) {
				continue;
			}
			mediaSuitability.value = { ...mediaSuitability.value, [m.url]: [] };
			try {
				await waitForClient();
				const res = await window.client.getMediaSuitability({ filename: m.filename });
				mediaSuitability.value = { ...mediaSuitability.value, [m.url]: res.networks || [] };
			} catch {
				// No indicator rather than a misleading one.
			}
		}
	});

	// Networks of the accounts chosen for this post; all networks before any are chosen.
	const selectedProtocols = computed(() => {
		const chosen = items.value.filter((a) => selectedAccountIds.value.includes(a.id)).map((a) => a.connector);
		return chosen.length ? [...new Set(chosen)] : null;
	});

	function relevantNetworks(media) {
		const networks = mediaSuitability.value[media.url] || [];
		const protocols = selectedProtocols.value;
		return protocols ? networks.filter((n) => protocols.includes(n.protocol)) : networks;
	}

	function mediaVerdict(media) {
		return worstVerdict(relevantNetworks(media).map((n) => n.verdict));
	}

	function mediaVerdictTitle(media) {
		return relevantNetworks(media).filter((n) => n.verdict !== 'ok').map(describeNetwork).join('\n');
	}

	// Library items by URL, for finding an attachment's optimised copy.
	const libraryByUrl = computed(() => new Map(mediaLibrary.value.map((item) => [item.url, item])));
	const libraryByName = computed(() => new Map(mediaLibrary.value.map((item) => [item.filename, item])));

	watch(selectedMedia, (media) => {
		if (media.length && mediaLibrary.value.length === 0) {
			loadMediaLibrary();
		}
	});

	// Attachments with an optimised copy that isn't attached yet; a copy made
	// for posting is preferred, as it fits every network.
	const optimisedSwaps = computed(() => selectedMedia.value.flatMap((m, index) => {
		const original = libraryByUrl.value.get(m.url);
		const copyName = original?.postingVersion || original?.optimisedVersion;
		const optimised = copyName && libraryByName.value.get(copyName);
		if (!optimised || selectedMedia.value.some((s) => s.url === optimised.url)) {
			return [];
		}
		return [{ index, original, optimised }];
	}));

	function useOptimised(swap) {
		const next = [...selectedMedia.value];
		next[swap.index] = swap.optimised;
		selectedMedia.value = next;
	}

	// Only judged against chosen accounts, so warnings match where the post goes.
	const mediaProblems = computed(() => {
		if (!selectedProtocols.value) {
			return [];
		}
		return selectedMedia.value.flatMap((m) => relevantNetworks(m)
			.filter((n) => n.verdict !== 'ok')
			.map((network) => ({ filename: m.filename, network })));
	});
	function applyMediaFromQuery() {
		const mediaUrl = route.query.mediaUrl;
		if (!mediaUrl || typeof mediaUrl !== 'string') {
			return;
		}
		const filename = mediaUrl.split('/').pop() || mediaUrl;
		const item = { url: mediaUrl, filename };
		if (selectedMedia.value.some((m) => m.url === item.url)) {
			return;
		}
		selectedMedia.value = [...selectedMedia.value, item];
	}

	function setComposerContent(content) {
		const text = typeof content === 'string' ? content : '';
		const textarea = postTextarea.value || document.getElementById('post');
		if (textarea) {
			textarea.value = text;
		}
		postLength.value = text.length;
		contentLength.value = text.length;
		if (postLengthCounter.value) {
			if (text.length > 280) {
				postLengthCounter.value.classList.add('bad');
			} else {
				postLengthCounter.value.classList.remove('bad');
			}
		}
	}

	function selectApplicableAccountIds(desiredIds, accounts) {
		const wanted = (desiredIds || []).map((id) => Number(id)).filter((id) => id > 0);
		if (!wanted.length) {
			return;
		}
		const list = accounts || items.value || [];
		if (!list.length) {
			pendingPrefillAccountIds = wanted;
			selectedAccountIds.value = [...wanted];
			return;
		}
		const available = new Set(list.map((a) => Number(a.id)));
		selectedAccountIds.value = wanted.filter((id) => available.has(id));
		pendingPrefillAccountIds = [];
	}

	async function applyComposePrefillFromQuery() {
		const flag = route.query.prefill;
		if (flag !== '1' && flag !== 'true') {
			return;
		}
		await nextTick();
		const prefill = takeComposePrefill();
		if (prefill?.content) {
			setComposerContent(prefill.content);
		}
		if (prefill?.socialAccountIds?.length) {
			selectApplicableAccountIds(prefill.socialAccountIds, items.value);
		}
		const nextQuery = { ...route.query };
		delete nextQuery.prefill;
		router.replace({ name: 'postBox', query: nextQuery });
	}
	function removeMedia(index) {
		selectedMedia.value = selectedMedia.value.filter((_, i) => i !== index);
	}

	function toggleEmojiPicker() {
		showEmojiPicker.value = !showEmojiPicker.value;
		if (showEmojiPicker.value) {
			showLlmToolsMenu.value = false;
			showMediaPicker.value = false;
		}
	}

	function toggleLlmToolsMenu() {
		showLlmToolsMenu.value = !showLlmToolsMenu.value;
		if (showLlmToolsMenu.value) {
			showEmojiPicker.value = false;
			showMediaPicker.value = false;
			showScheduleDialog.value = false;
		}
	}

	function onSelectLlmTool(toolId) {
		showLlmToolsMenu.value = false;
		runLlmTool(toolId);
	}

	function onDocumentPointerDown(event) {
		if (!showLlmToolsMenu.value) {
			return;
		}
		const menu = llmToolsMenuEl.value;
		if (!menu) {
			return;
		}
		if (menu.contains(event.target) || event.target.closest('.llm-tools-menu-trigger')) {
			return;
		}
		showLlmToolsMenu.value = false;
	}

	function onDocumentKeydown(event) {
		if (event.key === 'Escape' && showLlmToolsMenu.value) {
			showLlmToolsMenu.value = false;
		}
	}

	function insertEmoji(unicode) {
		const textarea = postTextarea.value;
		if (!textarea) {
			return;
		}
		insertAtCursor(textarea, unicode);
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
      document.addEventListener('pointerdown', onDocumentPointerDown, true);
      document.addEventListener('keydown', onDocumentKeydown);

        await refreshAccounts()
		await loadEnabledLlmTools()

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

		applyMediaFromQuery();
		await applyComposePrefillFromQuery();
    });

	onUnmounted(() => {
		document.removeEventListener('pointerdown', onDocumentPointerDown, true);
		document.removeEventListener('keydown', onDocumentKeydown);
		if (llmElapsedTimer) {
			clearInterval(llmElapsedTimer);
			llmElapsedTimer = null;
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

		applyMediaFromQuery();
		await applyComposePrefillFromQuery();
	})

	watch(
		() => route.query.prefill,
		async (value) => {
			if (value === '1' || value === 'true') {
				await applyComposePrefillFromQuery();
			}
		},
	)

    async function refreshAccounts() {
      const campaignsRet = await window.client.getCampaigns();
      campaigns.value = campaignsRet.campaigns || [];

      if (selectedCampaignId.value && selectedCampaignId.value !== 0) {
        await applyCampaignMembership(selectedCampaignId.value)
      }
    }

    function onAccountsLoaded(accounts) {
      items.value = accounts || []
      if (pendingPrefillAccountIds.length) {
        selectApplicableAccountIds(pendingPrefillAccountIds, items.value);
      }
    }

    async function onCampaignChange() {
      if (!selectedCampaignId.value || selectedCampaignId.value === 0) {
        selectedAccountIds.value = []
        return
      }
      await applyCampaignMembership(selectedCampaignId.value)
    }

    async function applyCampaignMembership(campaignId) {
      try {
        const res = await window.client.getCampaignSocialAccounts({ campaignId })
        selectedAccountIds.value = [...(res.socialAccountIds || [])]
      } catch (e) {
        console.error('Failed to fetch campaign memberships', e)
      }
    }

    const postMode = ref('live');

    import { showNotification } from './../javascript/notifications.js'

const canSubmit = computed(() => contentLength.value > 0 && (selectedAccountIds.value.length > 0 || saveAsCannedPost.value))
const canSaveDraft = computed(() => (contentLength.value > 0 || selectedMedia.value.length > 0) && !saveAsCannedPost.value)

const submitButtonLabel = computed(() => {
  if (scheduledAt.value && !saveAsCannedPost.value) {
    return t('section.postbox.postLater');
  }
  return t('section.postbox.postNow');
})

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
        return [...selectedAccountIds.value];
    }

    function saveDraft() {
        submitPost({ asDraft: true });
    }

    async function loadEnabledLlmTools() {
        llmToolsLoading.value = true;
        llmToolError.value = '';
        try {
            await waitForClient();
            const res = await window.client.listEnabledLlmTools({});
            enabledLlmTools.value = res.tools || [];
        } catch (e) {
            console.error(e);
            enabledLlmTools.value = [];
        } finally {
            llmToolsLoading.value = false;
        }
    }

    async function loadLlmToolRuns(postId) {
        if (!postId) {
            llmToolSessions.value = llmToolSessions.value.filter((s) => s.pending);
            return;
        }
        try {
            await waitForClient();
            const res = await window.client.listLlmToolRuns({ postId });
            const loaded = buildLlmToolSessions(res.runs || []);
            const pendingByTool = new Map(
                llmToolSessions.value.filter((s) => s.pending).map((s) => [s.toolId, s]),
            );
            const merged = loaded.map((session) => {
                const pending = pendingByTool.get(session.toolId);
                if (!pending) {
                    return session;
                }
                pendingByTool.delete(session.toolId);
                return {
                    ...session,
                    pending: true,
                    startedAt: pending.startedAt,
                    elapsedMs: pending.elapsedMs,
                    elapsedLabel: pending.elapsedLabel,
                };
            });
            for (const pending of pendingByTool.values()) {
                merged.unshift(pending);
            }
            llmToolSessions.value = merged;
        } catch (e) {
            console.error(e);
            llmToolError.value = e.message || 'Failed to load tool runs.';
        }
    }

    async function ensureDraftForToolRun(content) {
        if (draftPostId.value) {
            await window.client.updatePostContent({
                postId: draftPostId.value,
                content,
            });
            return draftPostId.value;
        }
        if (!draftEnsurePromise) {
            draftEnsurePromise = (async () => {
                const res = await window.client.submitPost({
                    content,
                    socialAccounts: [],
                    campaignId: selectedCampaignId.value || 0,
                    saveAsDraft: true,
                    mediaUrls: selectedMedia.value.map((m) => m.url),
                });
                const saved = (res.posts || []).find((p) => p.success && p.id);
                if (!saved?.id) {
                    throw new Error('Failed to save draft before running tool.');
                }
                draftPostId.value = saved.id;
                return saved.id;
            })().finally(() => {
                draftEnsurePromise = null;
            });
        }
        const postId = await draftEnsurePromise;
        await window.client.updatePostContent({ postId, content });
        return postId;
    }

    function formatElapsed(ms) {
        const totalSec = Math.floor(ms / 1000);
        const min = Math.floor(totalSec / 60);
        const sec = totalSec % 60;
        if (min > 0) {
            return `${min}:${String(sec).padStart(2, '0')}`;
        }
        return `${sec}s`;
    }

    function syncPendingElapsed() {
        const now = Date.now();
        let hasPending = false;
        llmToolSessions.value = llmToolSessions.value.map((session) => {
            if (!session.pending) {
                return session;
            }
            hasPending = true;
            const elapsedMs = Math.max(0, now - session.startedAt);
            return {
                ...session,
                elapsedMs,
                elapsedLabel: formatElapsed(elapsedMs),
            };
        });
        if (!hasPending && llmElapsedTimer) {
            clearInterval(llmElapsedTimer);
            llmElapsedTimer = null;
        }
    }

    function ensureElapsedTimer() {
        if (llmElapsedTimer) {
            return;
        }
        llmElapsedTimer = setInterval(syncPendingElapsed, 250);
    }

    function toolNameById(toolId) {
        return enabledLlmTools.value.find((t) => t.id === toolId)?.name || 'LLM tool';
    }

    function upsertSessionPending(toolId) {
        const startedAt = Date.now();
        const existingIdx = llmToolSessions.value.findIndex((s) => s.toolId === toolId);
        if (existingIdx >= 0) {
            const existing = llmToolSessions.value[existingIdx];
            const next = {
                ...existing,
                pending: true,
                startedAt,
                elapsedMs: 0,
                elapsedLabel: '0s',
            };
            const copy = [...llmToolSessions.value];
            copy.splice(existingIdx, 1);
            llmToolSessions.value = [next, ...copy];
            return;
        }
        llmToolSessions.value = [
            {
                toolId,
                toolName: toolNameById(toolId),
                pending: true,
                startedAt,
                elapsedMs: 0,
                elapsedLabel: '0s',
                history: [],
            },
            ...llmToolSessions.value,
        ];
    }

    function finishSessionReply(toolId, reply) {
        llmToolSessions.value = llmToolSessions.value.map((session) => {
            if (session.toolId !== toolId) {
                return session;
            }
            return {
                ...session,
                pending: false,
                elapsedLabel: '',
                history: [reply, ...(session.history || [])],
            };
        });
    }

    async function runLlmTool(toolId) {
        const content = (postTextarea.value?.value || document.getElementById('post')?.value || '').trim();
        if (!content) {
            llmToolError.value = 'Add post content before running a tool.';
            return;
        }
        const existing = llmToolSessions.value.find((s) => s.toolId === toolId);
        if (existing?.pending) {
            return;
        }
        llmToolError.value = '';
        upsertSessionPending(toolId);
        ensureElapsedTimer();

        try {
            await waitForClient();
            const postId = await ensureDraftForToolRun(content);
            const res = await window.client.runLlmTool({ postId, toolId });
            if (res.run) {
                finishSessionReply(toolId, replyFromApiRun(res.run));
            } else {
                await loadLlmToolRuns(postId);
            }
        } catch (e) {
            console.error(e);
            const message = e.message || 'Failed to run LLM tool.';
            const session = llmToolSessions.value.find((s) => s.toolId === toolId);
            const elapsedLabel = session
                ? formatElapsed(Math.max(0, Date.now() - session.startedAt))
                : '0s';
            finishSessionReply(toolId, {
                id: `local-error-${Date.now()}`,
                status: 'error',
                outputContent: '',
                errorMessage: message,
                connectionName: '',
                model: '',
                created: new Date().toISOString(),
                elapsedLabel,
            });
        } finally {
            syncPendingElapsed();
        }
    }

    function submitPost(options = {}) {
        const asDraft = Boolean(options.asDraft);
        const post = document.getElementById('post').value;
        const submit = document.getElementById('submit');
        const saveDraftBtn = document.getElementById('save-draft');

        if (post.length == 0 && selectedMedia.value.length === 0) {
            alert(asDraft ? "Add text or attach media to save a draft." : "Please enter a message.");
            return;
        }

        const originalSubmitText = submit.innerText;
        const isScheduled = Boolean(scheduledAt.value) && !saveAsCannedPost.value;
        submit.innerText = asDraft ? 'Saving…' : (isScheduled ? 'Scheduling…' : 'Submitting...');
        submit.disabled = true;
        if (saveDraftBtn) saveDraftBtn.disabled = true;

        const selectedServices = getSelectedPostingServices();

        // If no social accounts are selected OR canned post option is checked, create a canned post
        if (!asDraft && (selectedServices.length === 0 || saveAsCannedPost.value)) {
            window.client.createCannedPost({
                content: post,
                campaignId: selectedCampaignId.value,
            })
            .then((res) => {
                submit.innerText = originalSubmitText;
                submit.disabled = false;
                if (saveDraftBtn) saveDraftBtn.disabled = !canSaveDraft.value;

                // Clear the textarea and selected media
                document.getElementById('post').value = '';
                postLength.value = 0;
                contentLength.value = 0;
                selectedMedia.value = [];
                if (postLengthCounter.value) {
                    postLengthCounter.value.classList.remove('bad');
                }

                showNotification(
                    'success',
                    'Canned Post Created',
                    'Your post has been saved as a canned post.',
                    '/canned-posts',
                    'View canned posts',
                );
            })
            .catch((error) => {
                alert("Error creating canned post: " + error);
                submit.innerText = originalSubmitText;
                submit.disabled = false;
                if (saveDraftBtn) saveDraftBtn.disabled = !canSaveDraft.value;
            });
            return;
        }

         // Use `create(SubmitPostRequestSchema)` to create a new message.
        let req = {
            content: post,
            socialAccounts: selectedServices,
			campaignId: selectedCampaignId.value,
        }

        if (asDraft) {
            req.saveAsDraft = true;
        } else if (scheduledAt.value) {
            req.scheduledAt = scheduledAt.value;
        }
        if (selectedMedia.value.length) {
            req.mediaUrls = selectedMedia.value.map(m => m.url);
        }
        if (!asDraft && draftPostId.value) {
            req.fromDraftPostId = draftPostId.value;
        }

        console.log(req)

        window.client.submitPost(req)
            .then((res) => {
                submit.innerText = originalSubmitText;
                submit.disabled = false;
                if (saveDraftBtn) saveDraftBtn.disabled = !canSaveDraft.value;

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

                if (asDraft) {
                    const saved = (res.posts || []).filter((x) => x.success);
                    if (saved.length && saved[0].id) {
                        draftPostId.value = saved[0].id;
                        loadLlmToolRuns(draftPostId.value);
                    }
                    const savedCount = saved.length;
                    showNotification(
                        'success',
                        'Draft saved',
                        savedCount === 1
                            ? 'Your draft has been saved.'
                            : `${savedCount} drafts saved.`,
                        '/timeline',
                        'View timeline',
                    );
                    return;
                }

                draftPostId.value = 0;
                llmToolSessions.value = [];

                for (let x of res.posts) {
                    const statusClass = x.success ? 'success' : 'bad';

                    showNotification(
                        statusClass,
                        'Posted to ' + x.socialAccountIcon,
                        'Post complete.',
                        x.postUrl || null,
                        x.postUrl ? 'Open post' : undefined,
                    );
                }
            })
            .catch((error) => {
                alert(asDraft ? ("Error saving draft: " + error) : ("Error posting message: " + error));
                submit.innerText = originalSubmitText;
                submit.disabled = false;
                if (saveDraftBtn) saveDraftBtn.disabled = !canSaveDraft.value;
            });
    }

    function openScheduleDialog() {
        showScheduleDialog.value = true;
        scheduledAtDialog.value = scheduledAt.value || "";
        showEmojiPicker.value = false;
        showMediaPicker.value = false;
        showLlmToolsMenu.value = false;
    }

    function cancelScheduleDialog() {
        showScheduleDialog.value = false;
    }

    function confirmScheduleDialog() {
        scheduledAt.value = scheduledAtDialog.value;
        showScheduleDialog.value = false;
    }

    function clearScheduleDialog() {
        scheduledAtDialog.value = "";
    }

    function clearSchedule() {
        scheduledAt.value = "";
        scheduledAtDialog.value = "";
    }

	defineExpose({
		startPost,
	});
</script>

<style scoped>
    section.post-box.embedded {
		padding: 0;
		margin: 0;
	}

    .post-compose-stack {
		width: 100%;
		display: flex;
		flex-direction: column;
		gap: 0.75rem;
	}

    input[type="checkbox"] {
        display: none;
    }

	fieldset {
        align-items: center;
	}

	.post-composer {
		display: flex;
		flex-direction: column;
		gap: 0.75rem;
		width: 100%;
		grid-column: 1 / -1;
	}

	.post-composer-input {
		width: 100%;
		box-sizing: border-box;
		resize: vertical;
		min-height: 8em;
	}

	.post-composer-meta {
		display: flex;
		justify-content: space-between;
		align-items: flex-start;
		position: relative;
		margin-top: -0.35rem;
	}

	.post-composer-tools {
		position: relative;
		display: flex;
		align-items: center;
		gap: 0.35rem;
	}

	.llm-tools-menu-wrap {
		position: relative;
	}

	.llm-tools-menu {
		position: absolute;
		top: calc(100% + 0.35rem);
		left: 0;
		z-index: 1000;
		min-width: 14rem;
		max-width: 22rem;
		padding: 0.35rem;
		border-radius: 0.5rem;
		border: 1px solid var(--border-color, #ccc);
		background: var(--background-primary, #fff);
		box-shadow: 0 4px 16px rgba(0, 0, 0, 0.2);
		display: flex;
		flex-direction: column;
		gap: 0.15rem;
	}

	.llm-tools-menu-hint {
		margin: 0;
		padding: 0.5rem 0.65rem;
		font-size: 0.85rem;
		opacity: 0.75;
	}

	.llm-tools-menu-item {
		display: flex;
		flex-direction: column;
		align-items: flex-start;
		gap: 0.15rem;
		width: 100%;
		text-align: left;
		padding: 0.5rem 0.65rem;
		border: none;
		border-radius: 0.35rem;
		background: transparent;
		cursor: pointer;
		color: inherit;
		font: inherit;
	}

	.llm-tools-menu-item:hover:not(:disabled),
	.llm-tools-menu-item:focus-visible:not(:disabled) {
		background: var(--hover-background-color, #f0f0f0);
	}

	.llm-tools-menu-item:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}

	.llm-tools-menu-item-name {
		font-weight: 600;
	}

	.llm-tools-menu-item-desc {
		font-size: 0.85em;
		opacity: 0.75;
	}

	.composer-tool-btn {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		padding: 0.35rem 0.5rem;
	}

	.composer-tool-btn.active {
		background-color: var(--hover-background-color, #e8e8e8);
	}

	.post-length-counter {
		font-size: 0.85rem;
		opacity: 0.75;
		font-variant-numeric: tabular-nums;
	}

	.post-actions {
		grid-column: unset;
		flex-wrap: wrap;
		justify-content: flex-end;
		width: 100%;
		margin: 0;
	}

	.schedule-summary {
		display: flex;
		align-items: center;
		gap: 0.4rem;
		margin: 0;
		padding: 0.4rem 0.6rem;
		border-radius: 0.5rem;
		border: 1px solid var(--border-color, #ccc);
		background: var(--card-bg, #f8f8f8);
		font-size: 0.9rem;
	}

	.schedule-clear {
		margin-left: auto;
		width: 1.5rem;
		height: 1.5rem;
		padding: 0;
		border-radius: 50%;
		border: none;
		background: rgba(0, 0, 0, 0.08);
		cursor: pointer;
		font-size: 1.1rem;
		line-height: 1;
	}

	.media-attach {
		display: flex;
		flex-direction: column;
		gap: 0.5rem;
		padding-top: 0.25rem;
		border-top: 1px solid var(--border-color, #ddd);
	}

	.selected-media {
		display: flex;
		flex-wrap: wrap;
		gap: 0.5em;
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
	.media-verdict {
		position: absolute;
		left: 2px;
		bottom: 2px;
		display: inline-flex;
		padding: 2px;
		font-size: 0.9em;
	}
	.media-problem-verdict {
		display: inline-flex;
		align-items: center;
		gap: 0.1em;
		padding: 0.05em 0.35em;
		flex-shrink: 0;
	}
	.media-swaps {
		list-style: none;
		margin: 0;
		padding: 0;
		font-size: 0.85em;
		display: flex;
		flex-direction: column;
		gap: 0.3em;
	}
	.media-swaps li {
		display: flex;
		align-items: center;
		gap: 0.4em;
	}
	.media-swaps span {
		flex: 1;
	}
	.media-problems {
		list-style: none;
		margin: 0;
		padding: 0;
		font-size: 0.85em;
		display: flex;
		flex-direction: column;
		gap: 0.2em;
	}
	.media-problems li {
		display: flex;
		align-items: flex-start;
		gap: 0.3em;
	}
	.media-problems svg {
		flex-shrink: 0;
		margin-top: 0.15em;
	}
	.media-picker-item {
		position: relative;
	}
	.media-picker {
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

	.modal-overlay {
		position: fixed;
		top: 0;
		left: 0;
		width: 100%;
		height: 100%;
		background-color: rgba(0, 0, 0, 0.5);
		display: flex;
		justify-content: center;
		align-items: center;
		z-index: 1000;
	}

	.modal {
		background: var(--card-bg, white);
		padding: 2rem;
		border-radius: 0.5rem;
		box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1);
		min-width: 320px;
		max-width: 500px;
	}

	.modal h3 {
		margin: 0 0 1rem 0;
	}

	.modal label {
		display: block;
		margin-bottom: 0.5rem;
		font-weight: 500;
	}

	.modal input[type="datetime-local"] {
		width: 100%;
		box-sizing: border-box;
		margin-bottom: 1rem;
	}

	.dialog-actions {
		display: flex;
		flex-wrap: wrap;
		gap: 0.5rem;
		justify-content: flex-end;
	}
</style>
