-- 用户显示名（本地多角色切换）
ALTER TABLE users ADD COLUMN display_name TEXT NOT NULL DEFAULT '';
