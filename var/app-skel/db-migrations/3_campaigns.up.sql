CREATE TABLE campaigns (
	id int(10) not null primary key auto_increment,
	name VARCHAR(255) NOT NULL,
	description TEXT,
	start_date TIMESTAMP NOT NULL,
	end_date TIMESTAMP NOT NULL,
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- campaign_social_accounts is created in 6_campaign_social_accounts.up.sql (single canonical schema).

ALTER TABLE posts ADD COLUMN campaign_id int(10) DEFAULT NULL;
