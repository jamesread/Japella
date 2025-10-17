ALTER TABLE posts ADD COLUMN state varchar(32) NOT NULL DEFAULT 'completed' AFTER status;
ALTER TABLE posts ADD COLUMN scheduled_at datetime(3) DEFAULT NULL AFTER remote_id;
