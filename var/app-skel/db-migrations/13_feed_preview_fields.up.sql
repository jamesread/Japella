ALTER TABLE feed ADD COLUMN preview_url VARCHAR(1024) DEFAULT NULL AFTER remote_id;
ALTER TABLE feed ADD COLUMN preview_title VARCHAR(255) DEFAULT NULL AFTER preview_url;
ALTER TABLE feed ADD COLUMN preview_description TEXT DEFAULT NULL AFTER preview_title;
ALTER TABLE feed ADD COLUMN preview_image_url VARCHAR(1024) DEFAULT NULL AFTER preview_description;
