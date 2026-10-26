// Clickable hashtags in post content, opening the tag options dialog.

import { ref, computed } from 'vue';
import { connectorFromPost } from './util';

// A '#' not preceded by a word character, '&' (entities), '/' or another '#'
// (URL fragments), followed by a tag with at least one letter, so "#1" isn't one.
const HASHTAG_RE = /(^|[^\p{L}\p{N}_&/#])#([\p{L}\p{N}_]*[\p{L}_][\p{L}\p{N}_]*)/gu;
const TAG_PATH_RE = /\/(?:tags|hashtag|search\/hashtag)\/([^/?#]+)/i;

/** The hashtag's page on the network the post came from, or '' if unknown. */
export function hashtagUrl(post, tag) {
	const t = encodeURIComponent(tag);
	switch (connectorFromPost(post)) {
	case 'mastodon': {
		// The author's instance has a page for every tag its posts use.
		try {
			return `${new URL(post.remoteUrl || post.authorProfileUrl).origin}/tags/${t}`;
		} catch {
			return '';
		}
	}
	case 'bluesky':
		return `https://bsky.app/hashtag/${t}`;
	case 'x':
		return `https://x.com/hashtag/${t}`;
	default:
		return '';
	}
}

// Platform hashtag links (Mastodon sends <a class="hashtag" rel="tag">).
function hashtagFromLink(a) {
	const text = (a.textContent || '').trim();
	if (!text.startsWith('#')) {
		return '';
	}
	const isTag = a.classList.contains('hashtag')
		|| (a.getAttribute('rel') || '').split(/\s+/).includes('tag')
		|| TAG_PATH_RE.test(a.getAttribute('href') || '');
	return isTag ? text.slice(1) : '';
}

/**
 * Marks the hashtags in post HTML with data-hashtag, turning plain-text ones
 * into links to the tag's page. Content is only re-serialised, never run.
 */
export function linkifyHashtags(html, post) {
	if (!html || !html.includes('#') || typeof DOMParser === 'undefined') {
		return html;
	}
	const doc = new DOMParser().parseFromString(`<body>${html}</body>`, 'text/html');

	for (const a of doc.body.querySelectorAll('a')) {
		const tag = hashtagFromLink(a);
		if (tag) {
			a.dataset.hashtag = tag;
		}
	}

	const walker = doc.createTreeWalker(doc.body, NodeFilter.SHOW_TEXT);
	const textNodes = [];
	for (let node = walker.nextNode(); node; node = walker.nextNode()) {
		if (!node.parentElement?.closest('a, code, pre') && node.nodeValue.includes('#')) {
			textNodes.push(node);
		}
	}

	for (const node of textNodes) {
		const text = node.nodeValue;
		const frag = doc.createDocumentFragment();
		let last = 0;
		for (const m of text.matchAll(HASHTAG_RE)) {
			const start = m.index + m[1].length;
			frag.append(text.slice(last, start));
			const a = doc.createElement('a');
			const url = hashtagUrl(post, m[2]);
			a.href = url || '#';
			if (url) {
				a.target = '_blank';
				a.rel = 'noopener noreferrer';
			}
			a.className = 'hashtag';
			a.dataset.hashtag = m[2];
			a.textContent = `#${m[2]}`;
			frag.append(a);
			last = start + m[2].length + 1;
		}
		if (last > 0) {
			frag.append(text.slice(last));
			node.replaceWith(frag);
		}
	}

	return doc.body.innerHTML;
}

/**
 * Post content with clickable hashtags. Bind contentHtml with v-html and
 * onContentClick to the content element; tagOptions is the open dialog's
 * { tag, url }, or null.
 */
export function useHashtagOptions(getPost) {
	const tagOptions = ref(null);
	const contentHtml = computed(() => linkifyHashtags(getPost()?.content || '', getPost()));

	function onContentClick(e) {
		const link = e.target.closest?.('[data-hashtag]');
		// Modified clicks still open the link itself, e.g. in a new tab.
		if (!link || e.button !== 0 || e.ctrlKey || e.metaKey || e.shiftKey || e.altKey) {
			return;
		}
		e.preventDefault();
		const tag = link.dataset.hashtag;
		const href = link.getAttribute('href') || '';
		tagOptions.value = { tag, url: href && href !== '#' ? link.href : hashtagUrl(getPost(), tag) };
	}

	return { tagOptions, contentHtml, onContentClick };
}
