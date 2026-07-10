CREATE TABLE IF NOT EXISTS user_domain_profiles (
  user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  domain_id TEXT NOT NULL,
  summary TEXT NOT NULL DEFAULT '',
  updated_at TEXT NOT NULL,
  PRIMARY KEY (user_id, domain_id)
);

CREATE INDEX IF NOT EXISTS idx_user_domain_profiles_user ON user_domain_profiles(user_id);
