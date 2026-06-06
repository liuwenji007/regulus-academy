CREATE TABLE IF NOT EXISTS domain_build_jobs (
  id TEXT PRIMARY KEY,
  user_id TEXT NOT NULL,
  topic TEXT NOT NULL,
  goal TEXT NOT NULL DEFAULT '',
  force_build INTEGER NOT NULL DEFAULT 0,
  status TEXT NOT NULL,
  phase TEXT NOT NULL DEFAULT '',
  message TEXT NOT NULL DEFAULT '',
  result_json TEXT,
  error TEXT,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_domain_build_jobs_user_status
  ON domain_build_jobs (user_id, status, updated_at);
