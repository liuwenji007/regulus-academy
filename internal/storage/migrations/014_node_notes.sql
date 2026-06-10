CREATE TABLE IF NOT EXISTS node_notes (
    user_id    TEXT NOT NULL,
    domain_id  TEXT NOT NULL,
    node_key   TEXT NOT NULL,
    content_md TEXT NOT NULL DEFAULT '',
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, domain_id, node_key)
);
