ALTER TABLE domains ADD COLUMN tree_version INTEGER NOT NULL DEFAULT 1;

CREATE TABLE IF NOT EXISTS domain_extensions (
  id TEXT PRIMARY KEY,
  domain_id TEXT NOT NULL,
  user_id TEXT NOT NULL,
  from_version INTEGER NOT NULL,
  to_version INTEGER NOT NULL,
  added_nodes_json TEXT NOT NULL,
  reason TEXT,
  created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_domain_extensions_domain ON domain_extensions(domain_id);
