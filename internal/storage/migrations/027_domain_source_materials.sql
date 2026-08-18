-- 导入建课用的原始抽字正文（PDF / URL），供课程页回看，避免只剩模型摘要
CREATE TABLE IF NOT EXISTS domain_source_materials (
    domain_id TEXT PRIMARY KEY,
    kind TEXT NOT NULL,
    label TEXT NOT NULL DEFAULT '',
    body TEXT NOT NULL,
    page_count INTEGER NOT NULL DEFAULT 0,
    char_count INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (domain_id) REFERENCES domains(id)
);
