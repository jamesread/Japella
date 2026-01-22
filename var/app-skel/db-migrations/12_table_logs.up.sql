CREATE TABLE table_logs (
    id INT(10) UNSIGNED NOT NULL PRIMARY KEY AUTO_INCREMENT,
    message TEXT NOT NULL,
    level VARCHAR(50) NOT NULL,
    related_social_account_id INT(10) UNSIGNED DEFAULT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_table_logs_related_social_account (related_social_account_id),
    INDEX idx_table_logs_created_at (created_at),
    INDEX idx_table_logs_level (level),
    FOREIGN KEY (related_social_account_id) REFERENCES social_accounts(id) ON DELETE SET NULL
);
