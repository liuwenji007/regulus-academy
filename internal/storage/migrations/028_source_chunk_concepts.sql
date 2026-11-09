-- 召回块上的半结构概念（蒸馏大纲章节 concepts 的 JSON 数组）
ALTER TABLE source_chunks ADD COLUMN concepts_json TEXT NOT NULL DEFAULT '';
