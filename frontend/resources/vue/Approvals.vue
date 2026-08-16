<template>
	<Section
		title="Approvals"
		subtitle="Review posts waiting for approval. Open a post to edit content before approving; approve only when you are the current-stage assignee (and not the submitter)."
		classes="approvals"
		:padding="false"
	>
		<template #toolbar>
			<button type="button" class="neutral" title="Refresh" :disabled="loading" @click="loadPending">
				<Icon icon="material-symbols:refresh" />
			</button>
		</template>

		<div v-if="errorMessage" class="inline-notification error pad">{{ errorMessage }}</div>
		<div v-if="successMessage" class="inline-notification good pad">{{ successMessage }}</div>

		<Loading v-if="loading" message="Loading pending approvals…" :centered="true" />
	</Section>

	<p v-if="!loading && !pending.length" class="approvals-empty inline-notification note">
		No pending approval posts for you.
	</p>

	<section
		v-for="item in pending"
		v-else-if="!loading"
		:key="item.post?.id"
		class="small approval-entry"
		:data-approval-post-id="item.post?.id"
	>
		<PostPreview
			:post="item.post"
			timeline-mode
			@view-details="openPostDetails(item.post)"
		/>

		<div class="approval-meta">
			<span class="meta-pill">{{ item.accountPolicyName || `Policy #${item.accountPolicyId}` }}</span>
			<span class="meta-pill">Stage {{ (item.approvalStage ?? 0) + 1 }}</span>
			<span class="meta-pill">{{ sourceLabel(item.submissionSource) }}</span>
			<span class="meta-pill muted-pill">
				From {{ item.submittedByUsername || (item.submittedByUserId ? `#${item.submittedByUserId}` : 'unknown') }}
			</span>
			<span v-if="item.waitingOn" class="meta-pill waiting-pill">Waiting on {{ item.waitingOn }}</span>
		</div>

		<p v-if="!item.canApprove && !item.canReject" class="inline-notification note status-note">
			You can view this pending post but cannot act on it right now.
		</p>
		<p v-else-if="!item.canApprove && item.canReject" class="inline-notification note status-note">
			You cannot approve this post (submitters cannot approve their own submissions). You can withdraw it by rejecting.
		</p>

		<div role="toolbar" class="approval-actions">
			<router-link
				class="button inline-icon neutral"
				:to="{ name: 'postDetails', params: { id: String(item.post?.id) }, query: { from: 'approvals' } }"
			>
				<HugeiconsIcon
					:icon="EditIcon"
					width="1em"
					height="1em"
					:strokeWidth="iconStrokeWidth"
					aria-hidden="true"
				/>
				<span>Open / Edit</span>
			</router-link>
			<button
				type="button"
				class="inline-icon good"
				:disabled="acting === item.post?.id || !item.canApprove"
				:title="item.canApprove ? 'Approve current stage' : 'You cannot approve this post'"
				@click="openDialog(item, 'approve')"
			>
				<HugeiconsIcon
					v-if="acting === item.post?.id"
					:icon="Loading01Icon"
					width="1em"
					height="1em"
					:strokeWidth="iconStrokeWidth"
					aria-hidden="true"
				/>
				<HugeiconsIcon
					v-else
					:icon="CheckmarkCircle01Icon"
					width="1em"
					height="1em"
					:strokeWidth="iconStrokeWidth"
					aria-hidden="true"
				/>
				<span>{{ acting === item.post?.id ? 'Approving…' : 'Approve' }}</span>
			</button>
			<button
				type="button"
				class="inline-icon bad"
				:disabled="acting === item.post?.id || !item.canReject"
				:title="item.canReject ? 'Reject / withdraw' : 'You cannot reject this post'"
				@click="openDialog(item, 'reject')"
			>
				<HugeiconsIcon
					:icon="CancelCircleIcon"
					width="1em"
					height="1em"
					:strokeWidth="iconStrokeWidth"
					aria-hidden="true"
				/>
				<span>Reject</span>
			</button>
		</div>
	</section>

	<div v-if="dialogOpen" class="modal-overlay" @click.self="closeDialog">
		<div class="modal" role="dialog" :aria-labelledby="dialogTitleId">
			<h3 :id="dialogTitleId">{{ dialogMode === 'approve' ? 'Approve post' : 'Reject post' }}</h3>

			<p v-if="dialogItem" class="dialog-preview">
				<SocialAccountChip
					v-if="dialogItem.post?.socialAccountId"
					:social-account-id="dialogItem.post.socialAccountId"
					:identity="dialogItem.post.socialAccountIdentity || `Account #${dialogItem.post.socialAccountId}`"
					:icon="dialogItem.post.socialAccountIcon"
				/>
				<span v-else>{{ dialogItem.post?.socialAccountIdentity || `Account #${dialogItem.post?.socialAccountId}` }}</span>
				— stage {{ (dialogItem.approvalStage ?? 0) + 1 }}
			</p>
			<pre v-if="dialogItem" class="dialog-content">{{ dialogItem.post?.content }}</pre>

			<template v-if="dialogMode === 'reject'">
				<label class="reason-label" for="approval-reason">Reason (optional)</label>
				<textarea
					id="approval-reason"
					v-model="dialogReason"
					rows="3"
					placeholder="Optional rejection reason…"
					:disabled="acting > 0"
				/>
			</template>
			<p v-else class="dialog-confirm">Approve this post for the current stage?</p>

			<div role="toolbar" class="dialog-actions">
				<button type="button" class="neutral" :disabled="acting > 0" @click="closeDialog">Cancel</button>
				<button
					v-if="dialogMode === 'approve'"
					type="button"
					class="inline-icon good"
					:disabled="acting > 0"
					@click="confirmAction"
				>
					<HugeiconsIcon
						v-if="acting > 0"
						:icon="Loading01Icon"
						width="1em"
						height="1em"
						:strokeWidth="iconStrokeWidth"
						aria-hidden="true"
					/>
					<HugeiconsIcon
						v-else
						:icon="CheckmarkCircle01Icon"
						width="1em"
						height="1em"
						:strokeWidth="iconStrokeWidth"
						aria-hidden="true"
					/>
					<span>{{ acting > 0 ? 'Approving…' : 'Approve' }}</span>
				</button>
				<button
					v-else
					type="button"
					class="inline-icon bad"
					:disabled="acting > 0"
					@click="confirmAction"
				>
					<HugeiconsIcon
						v-if="acting > 0"
						:icon="Loading01Icon"
						width="1em"
						height="1em"
						:strokeWidth="iconStrokeWidth"
						aria-hidden="true"
					/>
					<HugeiconsIcon
						v-else
						:icon="CancelCircleIcon"
						width="1em"
						height="1em"
						:strokeWidth="iconStrokeWidth"
						aria-hidden="true"
					/>
					<span>{{ acting > 0 ? 'Rejecting…' : 'Reject' }}</span>
				</button>
			</div>
		</div>
	</div>
</template>

<script setup>
import { inject, onMounted, ref } from 'vue';
import { useRouter } from 'vue-router';
import { Icon } from '@iconify/vue';
import { HugeiconsIcon } from '@hugeicons/vue';
import {
	CancelCircleIcon,
	CheckmarkCircle01Icon,
	EditIcon,
	Loading01Icon,
} from '@hugeicons/core-free-icons';
import Section from 'picocrank/vue/components/Section.vue';
import Loading from './Loading.vue';
import PostPreview from './PostPreview.vue';
import SocialAccountChip from './SocialAccountChip.vue';

const iconStrokeWidth = 2.5;
const router = useRouter();
const refreshApprovalsNavCount = inject('refreshApprovalsNavCount', null);

const loading = ref(true);
const acting = ref(0);
const pending = ref([]);
const errorMessage = ref('');
const successMessage = ref('');

const dialogOpen = ref(false);
const dialogMode = ref('approve');
const dialogItem = ref(null);
const dialogReason = ref('');
const dialogTitleId = 'approval-dialog-title';

function sourceLabel(source) {
	if (source === 'mcp') return 'MCP';
	if (source === 'ui') return 'UI';
	return source || '—';
}

function openPostDetails(post) {
	if (!post?.id) return;
	router.push({ name: 'postDetails', params: { id: String(post.id) }, query: { from: 'approvals' } });
}

function openDialog(item, mode) {
	if (mode === 'approve' && !item.canApprove) return;
	if (mode === 'reject' && !item.canReject) return;
	dialogItem.value = item;
	dialogMode.value = mode;
	dialogReason.value = '';
	dialogOpen.value = true;
	errorMessage.value = '';
	successMessage.value = '';
}

function closeDialog() {
	if (acting.value > 0) return;
	dialogOpen.value = false;
	dialogItem.value = null;
	dialogReason.value = '';
}

async function loadPending() {
	loading.value = true;
	errorMessage.value = '';
	try {
		const res = await window.client.listPendingApprovals({});
		pending.value = res.pending || [];
		refreshApprovalsNavCount?.(pending.value.length);
	} catch (e) {
		console.error(e);
		errorMessage.value = e?.message || 'Failed to load pending approvals';
	} finally {
		loading.value = false;
	}
}

async function confirmAction() {
	const item = dialogItem.value;
	const id = item?.post?.id;
	if (!id) return;

	acting.value = id;
	errorMessage.value = '';
	successMessage.value = '';
	const reason = dialogReason.value.trim();

	try {
		if (dialogMode.value === 'approve') {
			const res = await window.client.approvePost({ postId: id });
			successMessage.value = res.standardResponse?.message || 'Approved';
		} else {
			const res = await window.client.rejectPost({ postId: id, reason });
			successMessage.value = res.standardResponse?.message || 'Rejected';
		}
		dialogOpen.value = false;
		dialogItem.value = null;
		dialogReason.value = '';
		await loadPending();
	} catch (e) {
		errorMessage.value = e?.message || (dialogMode.value === 'approve' ? 'Failed to approve' : 'Failed to reject');
	} finally {
		acting.value = 0;
	}
}

onMounted(loadPending);
</script>

<style scoped>
.pad {
	padding: 0.75rem 1rem;
}

.approvals-empty {
	margin: 0 0 1rem;
}

section.approval-entry {
	margin-bottom: 1rem;
	padding: 0;
}

.approval-meta {
	display: flex;
	flex-wrap: wrap;
	align-items: center;
	gap: 0.4rem;
	max-width: 800px;
	margin: -0.5rem auto 0.75rem;
	padding: 0 1rem;
}

.dialog-preview {
	margin: 0 0 0.5rem;
	font-size: 0.95rem;
	display: flex;
	flex-wrap: wrap;
	align-items: center;
	gap: 0.4rem;
}

.meta-pill {
	display: inline-block;
	padding: 0.15rem 0.55rem;
	border-radius: 999px;
	border: 1px solid var(--border-color, #ccc);
	font-size: 0.85rem;
}

.muted-pill {
	opacity: 0.8;
}

.waiting-pill {
	border-color: #c9a227;
}

.dialog-content {
	white-space: pre-wrap;
	word-break: break-word;
	margin: 0 0 1rem;
	padding: 0.75rem;
	border-radius: 0.35rem;
	background: rgba(127, 127, 127, 0.08);
	font-family: inherit;
	font-size: 0.95rem;
	line-height: 1.45;
	max-height: 16rem;
	overflow: auto;
}

.status-note {
	max-width: 800px;
	margin: 0 auto 0.75rem;
	padding-left: 1rem;
	padding-right: 1rem;
}

.approval-actions,
.dialog-actions {
	display: flex;
	gap: 0.5rem;
	flex-wrap: wrap;
	justify-content: flex-end;
	max-width: 800px;
	margin: 0 auto;
	padding: 0 1rem 0.5rem;
}

.modal-overlay {
	position: fixed;
	inset: 0;
	background: rgba(0, 0, 0, 0.5);
	display: flex;
	align-items: center;
	justify-content: center;
	z-index: 1000;
	padding: 1rem;
}

.modal {
	background: var(--surface, #fff);
	color: inherit;
	border-radius: 0.5rem;
	padding: 1.25rem;
	width: min(36rem, 100%);
	max-height: 90vh;
	overflow: auto;
	box-shadow: 0 8px 32px rgba(0, 0, 0, 0.2);
}

.modal h3 {
	margin: 0 0 0.75rem;
}

.dialog-confirm {
	margin: 0 0 1rem;
}

.reason-label {
	display: block;
	margin-bottom: 0.35rem;
	font-weight: 600;
}

.modal textarea {
	width: 100%;
	box-sizing: border-box;
	margin-bottom: 1rem;
	resize: vertical;
	min-height: 4.5rem;
	padding: 0.5rem;
	font: inherit;
}
</style>
