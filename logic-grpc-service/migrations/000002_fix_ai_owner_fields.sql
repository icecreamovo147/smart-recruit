-- 000002_fix_ai_owner_fields.sql
-- Adds owner_role/owner_id to ai_chat_sessions and ai_chat_history
-- for candidate AI support. Backfills existing HR-owned records.
-- MySQL DDL performs implicit commits, so this migration is intentionally
-- written as an ordered sequence rather than a transactional script.

ALTER TABLE ai_chat_sessions
  ADD COLUMN owner_role TINYINT NOT NULL DEFAULT 2 COMMENT '归属角色：1=候选人 2=HR' AFTER hr_id,
  ADD COLUMN owner_id BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '归属用户ID' AFTER owner_role,
  ADD KEY idx_owner_deleted_updated (owner_role, owner_id, deleted_at, updated_at);

ALTER TABLE ai_chat_history
  ADD COLUMN owner_role TINYINT NOT NULL DEFAULT 2 COMMENT '归属角色：1=候选人 2=HR' AFTER hr_id,
  ADD COLUMN owner_id BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '归属用户ID' AFTER owner_role,
  ADD KEY idx_owner_session_created (owner_role, owner_id, session_id, created_at);

-- Backfill existing HR sessions: set owner_role=2, owner_id=hr_id
UPDATE ai_chat_sessions SET owner_role = 2, owner_id = hr_id WHERE owner_role = 2 AND owner_id = 0 AND hr_id > 0;
UPDATE ai_chat_history SET owner_role = 2, owner_id = hr_id WHERE owner_role = 2 AND owner_id = 0 AND hr_id > 0;
