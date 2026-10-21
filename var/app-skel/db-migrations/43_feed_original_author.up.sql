ALTER TABLE feed
	ADD COLUMN original_author_id VARCHAR(255) NOT NULL DEFAULT '' AFTER author_avatar_url,
	ADD COLUMN original_author_name VARCHAR(255) NOT NULL DEFAULT '' AFTER original_author_id,
	ADD COLUMN original_author_avatar_url VARCHAR(1024) NOT NULL DEFAULT '' AFTER original_author_name;
