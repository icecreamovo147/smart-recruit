-- 000022_align_notification_indexes.sql
-- Replace receiver_role-based notification indexes with receiver_account_type
-- equivalents, so that all notification indexes use the current column.
-- MySQL DDL performs implicit commits, so this migration is intentionally
-- written as an ordered sequence rather than a transactional script.
ALTER TABLE notifications
  DROP INDEX idx_receiver_read_created,
  DROP INDEX idx_receiver_read_created_id,
  DROP INDEX idx_receiver_created,
  ADD INDEX idx_receiver_read_created
    (receiver_id, receiver_account_type, is_read, created_at),
  ADD INDEX idx_receiver_read_created_id
    (receiver_id, receiver_account_type, is_read, created_at, id),
  ADD INDEX idx_receiver_created
    (receiver_id, receiver_account_type, created_at);
