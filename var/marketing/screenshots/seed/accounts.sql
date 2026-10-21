-- Fictional social accounts for "Acme Roasters", a made-up coffee brand.
-- Real Mastodon/Bluesky/X accounts need OAuth, which the API can't fake, so
-- they are inserted directly. No tokens are stored (empty strings, as the
-- model can't scan NULLs), and the Makefile points the backend's outbound
-- HTTP at a closed port, so nothing ever contacts a real network.
-- owner_user_id 2 is the bootstrap admin user.
INSERT INTO social_accounts (id, created_at, updated_at, connector, identity, did, homeserver, active, state, owner_user_id, oauth2_token, oauth2_token_expiry, oauth2_refresh_token) VALUES
  (1, NOW() - INTERVAL 90 DAY, NOW(), 'mastodon',  'acmeroasters@mastodon.example.org', '', 'https://mastodon.example.org', 1, 'active', 2, '', NOW() + INTERVAL 60 DAY, ''),
  (2, NOW() - INTERVAL 90 DAY, NOW(), 'bluesky',   'acmeroasters.example.com', 'did:plc:demo-acmeroasters', 'https://bsky.example.com', 1, 'active', 2, '', NOW() + INTERVAL 60 DAY, ''),
  (3, NOW() - INTERVAL 80 DAY, NOW(), 'x',         'AcmeRoasters', '', '', 1, 'active', 2, '', NOW() + INTERVAL 60 DAY, ''),
  (4, NOW() - INTERVAL 70 DAY, NOW(), 'facebook',  'Acme Roasters', '', '', 1, 'active', 2, '', NOW() + INTERVAL 60 DAY, ''),
  (5, NOW() - INTERVAL 60 DAY, NOW(), 'instagram', 'acmeroasters', '', '', 1, 'active', 2, '', NOW() + INTERVAL 60 DAY, ''),
  (6, NOW() - INTERVAL 30 DAY, NOW(), 'mastodon',  'acme-events@mastodon.example.org', '', 'https://mastodon.example.org', 0, 'active', 2, '', NOW() + INTERVAL 60 DAY, '');
