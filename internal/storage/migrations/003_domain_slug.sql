-- 领域 slug，用于幂等创建
ALTER TABLE domains ADD COLUMN slug TEXT;
CREATE UNIQUE INDEX IF NOT EXISTS idx_domains_slug ON domains(slug) WHERE slug IS NOT NULL AND slug != '';
