// Actions on feed posts shared by the diagnostic dialog and the post context menu.

import { ref, computed } from 'vue';
import { ViewOffIcon, Tag01Icon, RefreshIcon } from '@hugeicons/core-free-icons';
import { canTagPost } from './postTags.js';

/**
 * Whether a post can be marked read: feed and feed column posts (not
 * published timeline posts) by someone else. Column posts aren't stored, so
 * they're identified by remote URL / id instead of a feed id.
 */
export function canMarkPostRead(post, { feedContext }) {
	return Boolean(feedContext
		&& post?.fromExternalAuthor
		&& (post.id || post.remoteUrl || post.remoteId));
}

/** Hides the post from the caller's feed and feed columns. */
export async function markPostRead(post) {
	await window.client.markFeedPostRead({
		id: post.id || 0,
		remoteUrl: post.remoteUrl || '',
		remoteId: post.remoteId || '',
	});
}

/**
 * Right-click menu for a post component. feedContext is a function returning
 * whether the post is shown in a feed (so feed-only actions apply). Emits
 * 'post-hidden' with the post when it's marked read. onEditTags, when given,
 * opens the post's tag editor.
 */
export function usePostContextMenu(getPost, feedContext, emit, { onEditTags = null } = {}) {
	const open = ref(false);
	const x = ref(0);
	const y = ref(0);
	const error = ref('');

	const items = computed(() => {
		const post = getPost();
		const entries = [];
		if (canMarkPostRead(post, { feedContext: feedContext() })) {
			entries.push({ id: 'mark-read', label: 'Mark as read', icon: ViewOffIcon, showCount: false });
		}
		if (onEditTags && feedContext() && canTagPost(post)) {
			entries.push({ id: 'tags', label: 'Tags…', icon: Tag01Icon, showCount: false });
		}
		if (feedContext() && post?.canRefresh) {
			entries.push({ id: 'refresh', label: 'Refresh', icon: RefreshIcon, showCount: false });
		}
		if (entries.length === 0) {
			entries.push({ id: 'none', label: 'No options', disabled: true, showCount: false });
		}
		return entries;
	});

	function onContextMenu(e) {
		// Leave the browser menu for links, form fields and selected text, where
		// it's useful. Dialogs opened from a post (comments, tags) sit inside its
		// element, so their right-clicks bubble here too and are left alone.
		if (e.target.closest('a, input, textarea, select, [contenteditable], dialog')
			|| String(window.getSelection?.() || '') !== '') {
			return;
		}
		e.preventDefault();
		x.value = e.clientX;
		y.value = e.clientY;
		error.value = '';
		open.value = true;
	}

	async function onSelect(item) {
		if (item.id === 'tags') {
			onEditTags?.();
			return;
		}
		if (item.id === 'refresh') {
			await refresh();
			return;
		}
		if (item.id !== 'mark-read') {
			return;
		}
		const post = getPost();
		try {
			await markPostRead(post);
			emit('post-hidden', post);
		} catch (e) {
			error.value = `Failed to mark as read: ${e.rawMessage || e.message || e}`;
			window.alert(error.value);
		}
	}

	// Reloads the post (content, likes, comment count) and emits
	// 'post-refetched' with the fresh copy.
	async function refresh() {
		const post = getPost();
		try {
			const res = await window.client.refreshPost({
				post: {
					socialAccountId: post.socialAccountId,
					feedPostId: post.id || 0,
					remoteId: post.remoteId || '',
					remoteUrl: post.remoteUrl || '',
					interactionRemoteId: post.interactionRemoteId || '',
				},
			});
			if (res.post) {
				emit('post-refetched', res.post);
			}
		} catch (e) {
			window.alert(`Failed to refresh post: ${e.rawMessage || e.message || e}`);
		}
	}

	return { open, x, y, items, onContextMenu, onSelect };
}
