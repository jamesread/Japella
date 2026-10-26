// Internal Japella tags on feed posts. Tags are private to the user and are
// never posted to any network.

import { reactive, ref } from 'vue';

export const TAG_QUERY_PREFIX = 'tag:';

// Tags saved this session, by post key, so every copy of a post (main feed and
// columns) shows the change without reloading.
const savedTags = reactive(new Map());

// Bumped whenever tags change, so tag columns know to reload.
export const tagsVersion = ref(0);

/** Matches the server's post key: the remote URL, else the remote id. */
export function postTagKey(post) {
	return post?.remoteUrl || post?.remoteId || '';
}

export function canTagPost(post) {
	return Boolean(post?.socialAccountId && postTagKey(post));
}

export function tagsFor(post) {
	const key = postTagKey(post);
	if (key && savedTags.has(key)) {
		return savedTags.get(key);
	}
	return post?.tags || [];
}

/** Lowercase without a leading '#', as the server stores tags. */
export function normaliseTag(raw) {
	return String(raw || '').trim().replace(/^#+/, '').toLowerCase();
}

export const TAG_PATTERN = /^[\p{L}\p{N}_-]{1,64}$/u;

/** Replaces the post's tags; returns them as stored. */
export async function savePostTags(post, tags) {
	const res = await window.client.setPostTags({
		post: {
			socialAccountId: post.socialAccountId,
			feedPostId: post.id || 0,
			remoteId: post.remoteId || '',
			remoteUrl: post.remoteUrl || '',
			interactionRemoteId: post.interactionRemoteId || '',
		},
		snapshot: post,
		tags,
	});
	const stored = res.tags || [];
	savedTags.set(postTagKey(post), stored);
	tagsVersion.value += 1;
	return stored;
}

/** The user's tags, most used first: [{ tag, posts }]. */
export async function listPostTags() {
	const res = await window.client.listPostTags({});
	return res.tags || [];
}
