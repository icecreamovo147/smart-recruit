-- Down: 000002_fix_ai_owner_fields

ALTER TABLE ai_chat_history
  DROP KEY idx_owner_session_created,
  DROP COLUMN owner_id,
  DROP COLUMN owner_role;

ALTER TABLE ai_chat_sessions
  DROP KEY idx_owner_deleted_updated,
  DROP COLUMN owner_id,
  DROP COLUMN owner_role;
