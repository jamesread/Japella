ALTER TABLE social_accounts ADD COLUMN did varchar(255) DEFAULT '' AFTER identity;
ALTER TABLE social_accounts ADD COLUMN homeserver varchar(255) DEFAULT '' AFTER did;
