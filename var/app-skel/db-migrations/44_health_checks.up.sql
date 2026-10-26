-- Issues found by health checks (github.com/jamesread/armature-health). Times are unix seconds.
-- At most one open (resolved_at IS NULL) row per (check_id, issue_key); a recurrence after
-- resolution inserts a new row.
CREATE TABLE IF NOT EXISTS health_issues (
    id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    check_id VARCHAR(191) NOT NULL,
    issue_key VARCHAR(191) NOT NULL,
    severity VARCHAR(16) NOT NULL,
    title VARCHAR(512) NOT NULL,
    detail TEXT NOT NULL,
    link VARCHAR(512) NOT NULL,
    first_seen BIGINT NOT NULL,
    last_seen BIGINT NOT NULL,
    occurrences INT NOT NULL DEFAULT 1,
    resolved_at BIGINT NULL,
    acknowledged_at BIGINT NULL,
    acknowledged_by BIGINT NULL,
    KEY idx_health_issues_check (check_id, issue_key, resolved_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Last run of each health check.
CREATE TABLE IF NOT EXISTS health_check_runs (
    check_id VARCHAR(191) NOT NULL PRIMARY KEY,
    last_run_at BIGINT NOT NULL,
    duration_ms BIGINT NOT NULL,
    status VARCHAR(16) NOT NULL,
    error TEXT NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
