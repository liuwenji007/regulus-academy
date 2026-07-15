CREATE TABLE IF NOT EXISTS user_domain_access (
    user_id TEXT NOT NULL,
    domain_id TEXT NOT NULL,
    node_key TEXT NOT NULL DEFAULT '',
    accessed_at DATETIME NOT NULL,
    PRIMARY KEY (user_id, domain_id)
);

CREATE INDEX IF NOT EXISTS idx_user_domain_access_time
    ON user_domain_access (user_id, accessed_at DESC);
