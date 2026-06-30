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
const YAML_PROTOCOLS = new Set(['telegram', 'discord']);

export function connectorUsesOauth(protocol) {
  return OAUTH_PROTOCOLS.has(protocol?.toLowerCase());
}

export function connectorUsesYamlConfig(protocol) {
  return YAML_PROTOCOLS.has(protocol?.toLowerCase());
}
