-- Down: 000006_add_performance_indexes

ALTER TABLE notifications
  DROP KEY idx_receiver_read_created_id;

ALTER TABLE applications
  DROP KEY idx_job_status_current;

ALTER TABLE ai_chat_history
  DROP KEY idx_session_created_id;

ALTER TABLE jobs
  DROP KEY idx_hr_created_id,
  DROP KEY idx_status_created_id;
