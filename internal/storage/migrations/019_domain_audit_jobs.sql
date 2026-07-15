ALTER TABLE domain_build_jobs ADD COLUMN job_kind TEXT NOT NULL DEFAULT 'build';
ALTER TABLE domain_build_jobs ADD COLUMN domain_id TEXT;

CREATE INDEX IF NOT EXISTS idx_domain_build_jobs_domain
  ON domain_build_jobs (domain_id, status, updated_at);
