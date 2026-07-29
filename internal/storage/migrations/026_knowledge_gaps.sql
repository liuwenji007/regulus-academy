-- 认知缺口账本：统一汇聚划词 / 错题 / 跳级缺口信号
CREATE TABLE IF NOT EXISTS knowledge_gaps (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL,
    domain_id TEXT NOT NULL DEFAULT '',
    node_key TEXT NOT NULL DEFAULT '',
    concept TEXT NOT NULL,
    source TEXT NOT NULL,
    hit_count INTEGER NOT NULL DEFAULT 1,
    severity REAL NOT NULL DEFAULT 1.0,
    matched_domain_id TEXT NOT NULL DEFAULT '',
    matched_node_key TEXT NOT NULL DEFAULT '',
    reason TEXT NOT NULL DEFAULT '',
    resolved_at DATETIME,
    last_hit_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id),
    UNIQUE (user_id, domain_id, concept, source)
);

CREATE INDEX IF NOT EXISTS idx_knowledge_gaps_user_open
    ON knowledge_gaps (user_id, resolved_at, severity DESC, last_hit_at DESC);

CREATE INDEX IF NOT EXISTS idx_knowledge_gaps_user_domain
    ON knowledge_gaps (user_id, domain_id, resolved_at);
