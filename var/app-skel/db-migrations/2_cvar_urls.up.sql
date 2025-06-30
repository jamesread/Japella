-- Default "" instead of NULL, because nullable strings are a pain in Go.
ALTER TABLE cvars ADD COLUMN external_url varchar(1024) default "";
ALTER TABLE cvars ADD COLUMN docs_url varchar(1024) default "";
