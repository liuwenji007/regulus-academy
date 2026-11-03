-- 课程挂载的原文与召回用小块（与 Distill 大块切分无关）
CREATE TABLE IF NOT EXISTS domain_sources (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    domain_id TEXT NOT NULL,
    kind TEXT NOT NULL,
    filename TEXT NOT NULL DEFAULT '',
    url TEXT NOT NULL DEFAULT '',
    text TEXT NOT NULL,
    byte_size INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_domain_sources_user_domain
    ON domain_sources (user_id, domain_id, created_at);

CREATE TABLE IF NOT EXISTS source_chunks (
    id TEXT PRIMARY KEY,
    source_id TEXT NOT NULL,
    domain_id TEXT NOT NULL,
    chunk_index INTEGER NOT NULL,
    text TEXT NOT NULL,
    node_key TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (source_id) REFERENCES domain_sources(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_source_chunks_domain
    ON source_chunks (domain_id, node_key);
