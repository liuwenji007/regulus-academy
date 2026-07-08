-- 行动助手规划会话
CREATE TABLE IF NOT EXISTS planning_sessions (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    phase TEXT NOT NULL DEFAULT 'intake',
    plan_json TEXT NOT NULL DEFAULT '{}',
    status TEXT NOT NULL DEFAULT 'active',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id)
);

CREATE INDEX IF NOT EXISTS idx_planning_sessions_user_status
    ON planning_sessions (user_id, status, updated_at DESC);

CREATE TABLE IF NOT EXISTS planning_messages (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id TEXT NOT NULL,
    role TEXT NOT NULL,
    content TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (session_id) REFERENCES planning_sessions(id)
);

CREATE INDEX IF NOT EXISTS idx_planning_messages_session
    ON planning_messages (session_id, id);
