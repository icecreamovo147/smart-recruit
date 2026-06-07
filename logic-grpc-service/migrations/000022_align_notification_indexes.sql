-- 000022_align_notification_indexes.sql
-- Replace receiver_role-based notification indexes with receiver_account_type
-- equivalents, so that all notification indexes use the current column.
-- MySQL DDL performs implicit commits, so this migration is intentionally
-- written as an ordered sequence rather than a transactional script.
-- Uses IF EXISTS for idempotency when db.sql is pre-imported.

ALTER TABLE notifications
  DROP INDEX IF EXISTS idx_receiver_read_created,
  DROP INDEX IF EXISTS idx_receiver_read_created_id,
  DROP INDEX IF EXISTS idx_receiver_created;

ALTER TABLE notifications
  DROP INDEX IF EXISTS idx_receiver_read_created,
  ADD INDEX idx_receiver_read_created
    (receiver_id, receiver_account_type, is_read, created_at);

ALTER TABLE notifications
  DROP INDEX IF EXISTS idx_receiver_read_created_id,
  ADD INDEX idx_receiver_read_created_id
    (receiver_id, receiver_account_type, is_read, created_at, id);

ALTER TABLE notifications
  DROP INDEX IF EXISTS idx_receiver_created,
  ADD INDEX idx_receiver_created
    (receiver_id, receiver_account_type, created_at);
