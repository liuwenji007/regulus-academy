-- 领域节点边界与来源（LLM 生成或 Skill 包）
ALTER TABLE domains ADD COLUMN nodes_json TEXT;
ALTER TABLE domains ADD COLUMN source TEXT DEFAULT 'skill_pack';
