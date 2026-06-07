-- 000006_add_performance_indexes.sql
-- Composite indexes for cursor pagination and high-frequency queries.
-- Uses IF EXISTS for idempotency when db.sql is pre-imported.

ALTER TABLE jobs
  DROP INDEX IF EXISTS idx_status_created_id,
  ADD INDEX idx_status_created_id (status, created_at, id),
  DROP INDEX IF EXISTS idx_hr_created_id,
  ADD INDEX idx_hr_created_id (hr_id, created_at, id);

ALTER TABLE ai_chat_history
  DROP INDEX IF EXISTS idx_session_created_id,
  ADD INDEX idx_session_created_id (session_id, created_at, id);

ALTER TABLE applications
  DROP INDEX IF EXISTS idx_job_status_current,
  ADD INDEX idx_job_status_current (job_id, status, is_current);

ALTER TABLE notifications
  DROP INDEX IF EXISTS idx_receiver_read_created_id,
  ADD INDEX idx_receiver_read_created_id (receiver_id, receiver_role, is_read, created_at, id);
