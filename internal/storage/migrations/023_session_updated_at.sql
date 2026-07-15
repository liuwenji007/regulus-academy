-- 侧栏「上一节学的课」按最近活动排序；旧会话用消息时间或 created_at 回填。
ALTER TABLE sessions ADD COLUMN updated_at DATETIME;

UPDATE sessions SET updated_at = COALESCE(
  (SELECT MAX(created_at) FROM session_messages WHERE session_id = sessions.id),
  created_at
);
