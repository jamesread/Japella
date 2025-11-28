CREATE TABLE feed (
    id INT NOT NULL PRIMARY KEY AUTO_INCREMENT,
    social_account_id INT(10) unsigned NOT NULL,
    content LONGTEXT,
    posted_date DATETIME NOT NULL,
    author_id varchar(255) NOT NULL,
    remote_url VARCHAR(1024) NOT NULL,
    remote_id varchar(255) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_feed_social_account (social_account_id),
    INDEX idx_feed_posted_date (posted_date),
    INDEX idx_feed_author (author_id),
    INDEX idx_feed_remote_id (remote_id),
    FOREIGN KEY (social_account_id) REFERENCES social_accounts(id) ON DELETE CASCADE
);
