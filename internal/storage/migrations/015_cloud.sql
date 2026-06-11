-- Cloud Demo：活跃统计、BYOK、配额与 Token 记账
ALTER TABLE users ADD COLUMN last_seen_at DATETIME;

CREATE TABLE IF NOT EXISTS user_llm_credentials (
    user_id TEXT PRIMARY KEY REFERENCES users(id),
    provider TEXT NOT NULL,
    api_key_encrypted TEXT NOT NULL,
    base_url TEXT,
    model TEXT,
    updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS llm_usage_daily (
    user_id TEXT NOT NULL,
    usage_date TEXT NOT NULL,
    message_count INTEGER NOT NULL DEFAULT 0,
    prompt_tokens INTEGER NOT NULL DEFAULT 0,
    completion_tokens INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (user_id, usage_date)
);

CREATE TABLE IF NOT EXISTS llm_token_usage (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL,
    usage_date TEXT NOT NULL,
    prompt_tokens INTEGER NOT NULL,
    completion_tokens INTEGER NOT NULL,
    total_tokens INTEGER NOT NULL,
    call_kind TEXT NOT NULL,
    billed_to TEXT NOT NULL DEFAULT 'platform',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_llm_token_usage_date ON llm_token_usage(usage_date);
CREATE INDEX IF NOT EXISTS idx_llm_token_usage_user ON llm_token_usage(user_id, usage_date);
