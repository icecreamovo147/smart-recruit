-- Down: 000022_align_notification_indexes
-- Restore the original receiver_role-based notification indexes.

ALTER TABLE notifications
  DROP INDEX idx_receiver_read_created,
  DROP INDEX idx_receiver_read_created_id,
  DROP INDEX idx_receiver_created,
  ADD KEY idx_receiver_read_created
    (receiver_id, receiver_role, is_read, created_at),
  ADD KEY idx_receiver_read_created_id
    (receiver_id, receiver_role, is_read, created_at, id),
  ADD KEY idx_receiver_created
    (receiver_id, receiver_role, created_at);
