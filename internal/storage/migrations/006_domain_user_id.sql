-- 课程归属用户（每人独立知识库）
ALTER TABLE domains ADD COLUMN user_id TEXT;
UPDATE domains SET user_id = 'default' WHERE user_id IS NULL OR user_id = '';
DROP INDEX IF EXISTS idx_domains_slug;
CREATE UNIQUE INDEX IF NOT EXISTS idx_domains_user_slug ON domains(user_id, slug) WHERE slug IS NOT NULL AND slug != '';
