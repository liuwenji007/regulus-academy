-- Cloud：建课日计数
ALTER TABLE llm_usage_daily ADD COLUMN build_count INTEGER NOT NULL DEFAULT 0;
