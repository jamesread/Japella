-- Posts created through the API in seed.sh all get "now" as their creation
-- time, so anything sorted by it (e.g. campaigns by last post) would come out
-- in a random order. Spread them over the last day, in seed order, instead.
UPDATE posts SET created_at = NOW() - INTERVAL (40 - id) HOUR, updated_at = NOW() - INTERVAL (40 - id) HOUR
WHERE id > 10;
