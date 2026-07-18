-- Down: 000003_add_idempotency_constraints

ALTER TABLE resumes
  DROP KEY idx_user_uploaded,
  DROP KEY uk_user_valid_resume,
  DROP COLUMN valid_key;

ALTER TABLE applications
  DROP KEY idx_user_applied,
  DROP KEY idx_job_current_applied,
  DROP KEY uk_active_application,
  DROP COLUMN active_key;
