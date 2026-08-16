import {
  TelegramIcon,
  DiscordIcon,
  BlueskyIcon,
  MastodonIcon,
  NewTwitterIcon,
  Facebook01Icon,
  InstagramIcon,
  WhatsappIcon,
  GridViewIcon,
} from '@hugeicons/core-free-icons';

export function waitForClient() {
  return new Promise((resolve, reject) => {
    const TIMEOUT_MS = 10_000

    const interval = setInterval(() => {
      if (window.client) {
        clearInterval(interval)
        clearTimeout(timeout)
        resolve(window.client)
      }
    }, 50)

    const timeout = setTimeout(() => {
      clearInterval(interval)
      reject(new Error('Client not found within timeout period'))
    }, TIMEOUT_MS)
  })
}

const CONNECTOR_DOCS_BASE = 'https://jamesread.github.io/Japella/connectors';

const CONNECTOR_DOCS = {
  telegram: `${CONNECTOR_DOCS_BASE}/telegram.html`,
  discord: `${CONNECTOR_DOCS_BASE}/discord.html`,
  mastodon: `${CONNECTOR_DOCS_BASE}/mastodon.html`,
  x: `${CONNECTOR_DOCS_BASE}/x.html`,
  bluesky: `${CONNECTOR_DOCS_BASE}/bluesky.html`,
};

const CONNECTOR_DOCS_INDEX = `${CONNECTOR_DOCS_BASE}/index.html`;

export function connectorDocsUrl(protocol) {
  return CONNECTOR_DOCS[protocol] || CONNECTOR_DOCS_INDEX;
}

const CONNECTOR_HUGEICONS = {
  telegram: TelegramIcon,
  discord: DiscordIcon,
  bluesky: BlueskyIcon,
  mastodon: MastodonIcon,
  x: NewTwitterIcon,
  facebook: Facebook01Icon,
  instagram: InstagramIcon,
  whatsapp: WhatsappIcon,
};

export function connectorHugeIcon(protocol) {
  return CONNECTOR_HUGEICONS[protocol?.toLowerCase()] || GridViewIcon;
}

const CONNECTOR_INTROS = {
  mastodon:
    'Connect a Mastodon account to publish statuses and media. You will sign in with OAuth on your Mastodon instance and grant Japella permission to post on your behalf.',
  x: 'Connect an X (Twitter) account using OAuth 2.0. Japella will request permission to post tweets and media from your account.',
  bluesky:
    'Connect a Bluesky account with OAuth. Japella uses your credentials to publish posts and media to the AT Protocol network.',
  telegram:
    'Telegram is configured by an administrator in YAML (bot token). It does not use OAuth for social accounts; accounts are managed through the bot configuration.',
  discord:
    'Discord is configured by an administrator in YAML (bot token). It supports chat bot features rather than OAuth-based social account linking.',
  facebook:
    'Facebook requires the connector to be started and OAuth credentials configured by an administrator before you can link an account.',
  instagram:
    'Instagram requires the connector to be started and OAuth credentials configured by an administrator before you can link an account.',
};

export function connectorIntroText(protocol) {
  return (
    CONNECTOR_INTROS[protocol?.toLowerCase()] ||
    'Review the connector details below before connecting your account.'
  );
}

const OAUTH_PROTOCOLS = new Set(['mastodon', 'x', 'bluesky', 'facebook', 'instagram']);
const YAML_PROTOCOLS = new Set();

export function connectorUsesOauth(protocol) {
  return OAUTH_PROTOCOLS.has(protocol?.toLowerCase());
}

export function connectorUsesYamlConfig(protocol) {
  return YAML_PROTOCOLS.has(protocol?.toLowerCase());
}

const CONNECTOR_DISPLAY_NAMES = {
  mastodon: 'Mastodon',
  x: 'X',
  bluesky: 'Bluesky',
  facebook: 'Facebook',
  instagram: 'Instagram',
};

const CONNECTOR_ICON_TO_PROTOCOL = {
  'mdi:mastodon': 'mastodon',
  'bi:twitter-x': 'x',
  'bi:bluesky': 'bluesky',
  'mdi:facebook': 'facebook',
  'mdi:instagram': 'instagram',
};

export function connectorDisplayName(protocol) {
  if (!protocol) {
    return 'Social Provider';
  }

  const key = String(protocol).toLowerCase();
  return CONNECTOR_DISPLAY_NAMES[key] || key.charAt(0).toUpperCase() + key.slice(1);
}

export function socialProviderNameFromPost(post) {
  if (post?.socialAccountConnector) {
    return connectorDisplayName(post.socialAccountConnector);
  }

  if (post?.connector) {
    return connectorDisplayName(post.connector);
  }

  const protocol = CONNECTOR_ICON_TO_PROTOCOL[post?.socialAccountIcon];
  return connectorDisplayName(protocol);
}

function parseDateString(dateString) {
  if (!dateString) {
    return null;
  }

  let date = new Date(dateString.replace(' ', 'T'));
  if (isNaN(date.getTime())) {
    date = new Date(dateString);
  }
  if (isNaN(date.getTime())) {
    return null;
  }

  return date;
}

function formatTimeOfDay(date) {
  const hours = date.getHours();
  const minutes = date.getMinutes();
  const ampm = hours >= 12 ? 'pm' : 'am';
  const displayHours = hours % 12 || 12;

  if (minutes > 0) {
    return `${displayHours}:${minutes.toString().padStart(2, '0')}${ampm}`;
  }

  return `${displayHours}${ampm}`;
}

export function formatHumanDateTime(dateString) {
  const date = parseDateString(dateString);
  if (!date) {
    return dateString || '';
  }

  const now = new Date();
  const today = new Date(now.getFullYear(), now.getMonth(), now.getDate());
  const targetDate = new Date(date.getFullYear(), date.getMonth(), date.getDate());
  const diffDays = Math.round((targetDate - today) / (1000 * 60 * 60 * 24));
  const timeStr = formatTimeOfDay(date);
  const dayNames = ['Sunday', 'Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday'];
  const monthNames = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec'];

  if (diffDays === 0) {
    return `Today, ${timeStr}`;
  }
  if (diffDays === -1) {
    return `Yesterday, ${timeStr}`;
  }
  if (diffDays === 1) {
    return `Tomorrow, ${timeStr}`;
  }
  if (diffDays >= -6 && diffDays <= -2) {
    return `${dayNames[date.getDay()]}, ${timeStr}`;
  }
  if (diffDays >= 2 && diffDays <= 6) {
    return `${dayNames[date.getDay()]}, ${timeStr}`;
  }

  const month = monthNames[date.getMonth()];
  const day = date.getDate();

  if (date.getFullYear() === now.getFullYear()) {
    return `${month} ${day}, ${timeStr}`;
  }

  return `${month} ${day}, ${date.getFullYear()}, ${timeStr}`;
}

export function normalizeConnectorIssues(issues) {
  return (issues || []).map((issue) => {
    if (typeof issue === 'string') {
      return { message: issue };
    }

    return {
      message: issue.message || '',
      fixPath: issue.fixPath || '',
      fixHash: issue.fixHash || '',
      fixLabel: issue.fixLabel || '',
      fixAction: issue.fixAction || '',
    };
  });
}

export function connectorIssueFixRoute(issue) {
  if (!issue?.fixPath) {
    return null;
  }

  const route = { path: issue.fixPath };
  if (issue.fixHash) {
    route.hash = `#${issue.fixHash}`;
  }
  return route;
}

export function connectorIssueSummary(issues) {
  return normalizeConnectorIssues(issues)
    .map((issue) => issue.message)
    .join('; ');
}

export function connectorHasBlockingIssues(issues) {
  return normalizeConnectorIssues(issues).length > 0;
}

export const CONNECTOR_FIX_ACTION_REGISTER_CLIENT = 'register_client';

/** Decode HTML entities (e.g. &amp;, &raquo;) for plain-text display. */
export function decodeHtmlEntities(text) {
	if (!text) {
		return '';
	}
	if (typeof document === 'undefined') {
		return text;
	}

	const el = document.createElement('div');
	let current = text;
	for (let i = 0; i < 3; i++) {
		el.innerHTML = current;
		const next = el.textContent || '';
		if (next === current) {
			break;
		}
		current = next;
	}
	return current;
}
