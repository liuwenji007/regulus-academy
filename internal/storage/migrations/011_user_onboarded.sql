ALTER TABLE users ADD COLUMN onboarded_at DATETIME;
-- 已有画像的老用户视为已完成引导，避免重复弹窗
UPDATE users SET onboarded_at = created_at
WHERE COALESCE(profile_summary, '') != '' AND onboarded_at IS NULL;
