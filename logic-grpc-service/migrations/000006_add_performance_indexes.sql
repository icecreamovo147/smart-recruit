-- 000006_add_performance_indexes.sql
-- Composite indexes for cursor pagination and high-frequency queries.

ALTER TABLE jobs
  ADD KEY idx_status_created_id (status, created_at, id),
  ADD KEY idx_hr_created_id (hr_id, created_at, id);

ALTER TABLE ai_chat_history
  ADD KEY idx_session_created_id (session_id, created_at, id);

ALTER TABLE applications
  ADD KEY idx_job_status_current (job_id, status, is_current);

ALTER TABLE notifications
  ADD KEY idx_receiver_read_created_id (receiver_id, receiver_role, is_read, created_at, id);
