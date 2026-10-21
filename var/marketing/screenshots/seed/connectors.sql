-- Throwaway OAuth client credentials for the demo connectors. Without a
-- client id, starting the Mastodon connector tries to register an app on
-- mastodon.social; these placeholders stop that. They are not real values.
INSERT INTO cvars (key_name, title, value_string, value_int, description, default_value, category, type, created_at, updated_at) VALUES
  ('mastodon.client_id', 'Mastodon Client ID', 'demo-client-id', 0, '', '', 'Mastodon', 'text', NOW(), NOW()),
  ('mastodon.client_secret', 'Mastodon Client Secret', 'demo-client-secret', 0, '', '', 'Mastodon', 'password', NOW(), NOW());
