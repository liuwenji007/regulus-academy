-- 学习旁路：侧边对话消息 + 术语卡片缓存
CREATE TABLE IF NOT EXISTS aside_messages (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL,
    domain_id TEXT NOT NULL DEFAULT '',
    node_key TEXT NOT NULL DEFAULT '',
    coach_session_id TEXT NOT NULL DEFAULT '',
    role TEXT NOT NULL,
    content TEXT NOT NULL,
    anchor_text TEXT NOT NULL DEFAULT '',
    intent TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id)
);

CREATE INDEX IF NOT EXISTS idx_aside_messages_user_domain
    ON aside_messages (user_id, domain_id, id DESC);

CREATE TABLE IF NOT EXISTS term_cards (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL,
    domain_id TEXT NOT NULL DEFAULT '',
    node_key TEXT NOT NULL DEFAULT '',
    normalized_term TEXT NOT NULL,
    original_text TEXT NOT NULL DEFAULT '',
    card_json TEXT NOT NULL DEFAULT '{}',
    hit_count INTEGER NOT NULL DEFAULT 1,
    last_hit_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id),
    UNIQUE (user_id, domain_id, normalized_term)
);

CREATE INDEX IF NOT EXISTS idx_term_cards_user_domain_hit
    ON term_cards (user_id, domain_id, last_hit_at DESC);
