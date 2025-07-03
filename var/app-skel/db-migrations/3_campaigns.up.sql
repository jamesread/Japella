CREATE TABLE campaigns (
	id int(10) not null primary key auto_increment,
	name VARCHAR(255) NOT NULL,
	description TEXT,
	start_date TIMESTAMP NOT NULL,
	end_date TIMESTAMP NOT NULL,
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE campaign_social_accounts (
	id int(10) not null primary key auto_increment,
	campaign_id int(10) NOT NULL,
	social_account_group_id int(10) NOT NULL,
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

ALTER TABLE posts ADD COLUMN campaign_id int(10) DEFAULT NULL;
