-- 个性化裁剪：domains 表记录引用的公共知识树信息
ALTER TABLE domains ADD COLUMN ref_slug TEXT;
ALTER TABLE domains ADD COLUMN ref_version INTEGER;
ALTER TABLE domains ADD COLUMN selection_json TEXT;
