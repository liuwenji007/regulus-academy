-- Phase 2: 会话阶段与上下文
ALTER TABLE sessions ADD COLUMN phase TEXT NOT NULL DEFAULT 'explain';
ALTER TABLE sessions ADD COLUMN context_json TEXT NOT NULL DEFAULT '{}';
ALTER TABLE sessions ADD COLUMN domain_slug TEXT NOT NULL DEFAULT '';
