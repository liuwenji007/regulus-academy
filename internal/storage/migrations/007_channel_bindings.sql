CREATE TABLE IF NOT EXISTS channel_bindings (
  platform TEXT NOT NULL,
  platform_user_id TEXT NOT NULL,
  user_id TEXT NOT NULL REFERENCES users(id),
  display_name_snapshot TEXT,
  created_at TEXT NOT NULL,
  PRIMARY KEY (platform, platform_user_id)
);

CREATE TABLE IF NOT EXISTS channel_active_node (
  user_id TEXT PRIMARY KEY REFERENCES users(id),
  domain_id TEXT NOT NULL,
  node_key TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_channel_bindings_user ON channel_bindings(user_id);
